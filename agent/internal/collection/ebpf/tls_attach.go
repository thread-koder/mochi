package ebpf

import (
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"golang.org/x/arch/x86/x86asm"
)

var opensslHooks = []tlsHook{
	{"SSL_write", tlsCookieSSLWrite},
	{"SSL_read", tlsCookieSSLRead},
	{"SSL_write_ex", tlsCookieSSLWriteEx},
	{"SSL_read_ex", tlsCookieSSLReadEx},
}

var goTLSHooks = []tlsHook{
	{"crypto/tls.(*Conn).Write", tlsCookieGoWrite},
	{"crypto/tls.(*Conn).Read", tlsCookieGoRead},
}

func (c *Collector) attachFile(openPath string, allowGo bool) ([]io.Closer, bool, error) {
	var links []io.Closer
	hit := false

	if allowGo {
		goHit, err := c.attachGoTLS(openPath, &links)
		if err != nil {
			return nil, false, errors.Join(err, closeLinks(links))
		}
		hit = goHit
	} else {
		ex, err := link.OpenExecutable(openPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, false, nil
			}
			return nil, false, fmt.Errorf("open executable: %w", err)
		}
		sslHit, err := c.attachOpenSSL(ex, &links)
		if err != nil {
			return nil, false, errors.Join(err, closeLinks(links))
		}
		hit = sslHit
	}
	if !hit {
		return nil, false, closeLinks(links)
	}
	return links, true, nil
}

func (c *Collector) attachOpenSSL(ex *link.Executable, links *[]io.Closer) (bool, error) {
	hit := false
	for _, hook := range opensslHooks {
		ok, err := c.attachUprobe(ex, hook.symbol, c.tlsObjs.MochiTlsEnter, hook.cookie, 0, 0, false, links)
		if err != nil {
			return false, err
		}
		if !ok {
			continue
		}
		retOK, err := c.attachUprobe(ex, hook.symbol, c.tlsObjs.MochiTlsRet, hook.cookie, 0, 0, true, links)
		if err != nil {
			return false, err
		}
		if !retOK {
			return false, fmt.Errorf("attach %s return: symbol disappeared", hook.symbol)
		}
		hit = true
	}
	return hit, nil
}

func (c *Collector) attachGoTLS(openPath string, links *[]io.Closer) (bool, error) {
	if runtime.GOARCH != "amd64" {
		return false, nil
	}
	file, err := elf.Open(openPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("open ELF: %w", err)
	}
	defer file.Close()

	ex, err := link.OpenExecutable(openPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("open executable: %w", err)
	}

	hit := false
	for _, hook := range goTLSHooks {
		sym, ok := findFunc(file, hook.symbol)
		if !ok {
			continue
		}
		base, code, ok := symbolLoad(file, sym)
		if !ok {
			continue
		}
		offsets, ok := decodeRets(code)
		if !ok || len(offsets) == 0 {
			continue
		}
		attached, err := c.attachUprobe(ex, "", c.tlsObjs.MochiTlsEnter, hook.cookie, base, 0, false, links)
		if err != nil {
			return false, err
		}
		if !attached {
			continue
		}
		for _, off := range offsets {
			retAttached, err := c.attachUprobe(ex, "", c.tlsObjs.MochiTlsRet, tlsCookieGoRet, base, off, false, links)
			if err != nil {
				return false, err
			}
			if !retAttached {
				return false, fmt.Errorf("attach %s return at %d: symbol disappeared", hook.symbol, off)
			}
		}
		hit = true
	}
	return hit, nil
}

func (c *Collector) attachUprobe(ex *link.Executable, symbol string, prog *ebpf.Program, cookie, address, offset uint64, retprobe bool, links *[]io.Closer) (bool, error) {
	opts := &link.UprobeOptions{Cookie: cookie, Address: address, Offset: offset}
	var (
		lnk link.Link
		err error
	)
	if retprobe {
		lnk, err = ex.Uretprobe(symbol, prog, opts)
	} else {
		lnk, err = ex.Uprobe(symbol, prog, opts)
	}
	if err != nil {
		if missingSymbol(err) {
			return false, nil
		}
		kind := "uprobe"
		if retprobe {
			kind = "uretprobe"
		}
		name := symbol
		if name == "" {
			name = fmt.Sprintf("%#x", address+offset)
		}
		return false, fmt.Errorf("attach %s %s: %w", kind, name, err)
	}
	*links = append(*links, lnk)
	return true, nil
}

func findFunc(file *elf.File, name string) (elf.Symbol, bool) {
	if sym, ok := findFuncIn(file.Symbols, name); ok {
		return sym, true
	}
	return findFuncIn(file.DynamicSymbols, name)
}

func findFuncIn(load func() ([]elf.Symbol, error), name string) (elf.Symbol, bool) {
	syms, err := load()
	if err != nil {
		return elf.Symbol{}, false
	}
	for _, sym := range syms {
		if sym.Name == name && elf.ST_TYPE(sym.Info) == elf.STT_FUNC && sym.Size > 0 && sym.Value != 0 {
			return sym, true
		}
	}
	return elf.Symbol{}, false
}

// symbolLoad finds the executable PT_LOAD for sym and returns its file offset
// plus the function bytes (for RET decode). Uprobe Address is a file offset:
// Value - p_vaddr + p_offset.
func symbolLoad(file *elf.File, sym elf.Symbol) (fileOff uint64, code []byte, ok bool) {
	for _, prog := range file.Progs {
		if prog.Type != elf.PT_LOAD || prog.Flags&elf.PF_X == 0 {
			continue
		}
		if sym.Value < prog.Vaddr || sym.Value >= prog.Vaddr+prog.Memsz {
			continue
		}
		fileOff = sym.Value - prog.Vaddr + prog.Off
		reader := prog.Open()
		skip := int64(sym.Value - prog.Vaddr)
		if _, err := io.CopyN(io.Discard, reader, skip); err != nil {
			return 0, nil, false
		}
		code = make([]byte, sym.Size)
		if _, err := io.ReadFull(reader, code); err != nil {
			return 0, nil, false
		}
		return fileOff, code, true
	}
	return 0, nil, false
}

func decodeRets(code []byte) ([]uint64, bool) {
	var offsets []uint64
	for pc := 0; pc < len(code); {
		inst, err := x86asm.Decode(code[pc:], 64)
		if err != nil || inst.Len <= 0 {
			return nil, false
		}
		if inst.Op == x86asm.RET {
			offsets = append(offsets, uint64(pc))
		}
		pc += inst.Len
	}
	return offsets, true
}
