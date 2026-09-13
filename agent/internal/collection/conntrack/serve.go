package conntrack

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mdlayher/netlink"
	"github.com/thread_koder/mochi/agent/internal/logger"
	"github.com/ti-mo/conntrack"
	"github.com/ti-mo/netfilter"
)

var (
	errEventStreamDead = errors.New("conntrack event stream dead")
	errClientClosed    = errors.New("conntrack client closed")
)

type dumpResult struct {
	flows []conntrack.Flow
	err   error
}

func (c *Client) serveDumpOnly(ctx context.Context, ready chan<- struct{}) {
	defer close(c.serveDone)

	log := logger.WithComponent("conntrack")
	if err := c.runDumpGC(ctx, nil, nil); err != nil && !isShutdownErr(ctx, c, err) {
		log.Error().Err(err).Msg("Initial conntrack dump failed")
	}
	close(ready)

	ticker := time.NewTicker(eventsOffGCInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closing:
			return
		case <-ticker.C:
			if err := c.runDumpGC(ctx, nil, nil); err != nil && !isShutdownErr(ctx, c, err) {
				log.Error().Err(err).Msg("Conntrack dump GC failed")
			}
		}
	}
}

func (c *Client) serveEvents(ctx context.Context, ready chan<- struct{}) {
	defer close(c.serveDone)

	log := logger.WithComponent("conntrack")
	first := true
	signalReady := func() {
		if first {
			first = false
			close(ready)
		}
	}
	defer signalReady()

	for {
		if ctx.Err() != nil || c.isClosing() {
			return
		}

		evChan, errChan, err := c.openListen()
		if err != nil {
			log.Error().Err(err).Msg("Failed to listen for conntrack events")
			signalReady()
			select {
			case <-ctx.Done():
				return
			case <-c.closing:
				return
			case <-time.After(listenRetryInterval):
				continue
			}
		}

		if err := c.runDumpGC(ctx, evChan, errChan); err != nil {
			if isShutdownErr(ctx, c, err) {
				c.closeEvents(evChan, errChan)
				return
			}
			if errors.Is(err, errEventStreamDead) {
				log.Error().Err(err).Msg("Conntrack event stream lost during dump")
				c.closeEvents(evChan, errChan)
				signalReady()
				continue
			}
			log.Error().Err(err).Msg("Initial conntrack dump failed")
		}
		signalReady()

		if reconnect := c.serveListen(ctx, evChan, errChan); !reconnect {
			c.closeEvents(evChan, errChan)
			return
		}
		c.closeEvents(evChan, errChan)
	}
}

func (c *Client) openListen() (chan conntrack.Event, <-chan error, error) {
	events, err := conntrack.Dial(&netlink.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("dial conntrack events: %w", err)
	}
	if err := events.SetReadBuffer(eventReadBuffer); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("set conntrack read buffer: %w", err), events.Close())
	}

	evChan := make(chan conntrack.Event, eventChannelSize)
	errChan, err := events.Listen(evChan, eventListenWorkers, netfilter.GroupsCT)
	if err != nil {
		return nil, nil, errors.Join(fmt.Errorf("listen conntrack: %w", err), events.Close())
	}

	c.socketMu.Lock()
	c.events = events
	c.socketMu.Unlock()
	return evChan, errChan, nil
}

// serveListen applies events until shutdown or the Listen worker dies.
// reconnect is true when the event stream must be redialed.
func (c *Client) serveListen(ctx context.Context, evChan <-chan conntrack.Event, errChan <-chan error) (reconnect bool) {
	log := logger.WithComponent("conntrack")
	ticker := time.NewTicker(eventsGCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-c.closing:
			return false
		case ev := <-evChan:
			c.applyEvent(ev)
		case err, ok := <-errChan:
			if ok && err != nil {
				log.Error().Err(err).Msg("Conntrack event worker failed")
			}
			return true
		case <-ticker.C:
			if err := c.runDumpGC(ctx, evChan, errChan); err != nil {
				if isShutdownErr(ctx, c, err) {
					return false
				}
				if errors.Is(err, errEventStreamDead) {
					log.Error().Err(err).Msg("Conntrack event stream lost during dump GC")
					return true
				}
				log.Error().Err(err).Msg("Conntrack dump GC failed")
			}
		}
	}
}

func (c *Client) runDumpGC(ctx context.Context, evChan <-chan conntrack.Event, errChan <-chan error) error {
	c.cacheMu.Lock()
	startGeneration := c.generation
	c.generation++
	c.cacheMu.Unlock()

	c.socketMu.Lock()
	dump := c.dump
	c.socketMu.Unlock()
	if dump == nil {
		return fmt.Errorf("conntrack dump closed")
	}

	resultCh := make(chan dumpResult, 1)
	go func() {
		flows, err := dump.Dump(nil)
		if err != nil {
			resultCh <- dumpResult{err: fmt.Errorf("dump conntrack: %w", err)}
			return
		}
		resultCh <- dumpResult{flows: flows}
	}()

	var pending []conntrack.Event
	for {
		select {
		case <-ctx.Done():
			<-resultCh
			return context.Cause(ctx)
		case <-c.closing:
			<-resultCh
			return errClientClosed
		case ev := <-evChan:
			pending = append(pending, ev)
		case err, ok := <-errChan:
			<-resultCh
			if ok && err != nil {
				return fmt.Errorf("%w: %w", errEventStreamDead, err)
			}
			return errEventStreamDead
		case res := <-resultCh:
			if res.err != nil {
				return res.err
			}
			c.applyDumpGC(startGeneration, res.flows)
			for _, ev := range pending {
				c.applyEvent(ev)
			}
			return nil
		}
	}
}

// closeEvents closes the event Conn while draining evChan/errChan so ti-mo's
// Close Wait cannot deadlock on a worker blocked in evChan send. Silent close
// neither sends errChan nor closes evChan.
func (c *Client) closeEvents(evChan <-chan conntrack.Event, errChan <-chan error) {
	c.socketMu.Lock()
	events := c.events
	c.events = nil
	c.socketMu.Unlock()
	if events == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		if err := events.Close(); err != nil {
			log := logger.WithComponent("conntrack")
			log.Error().Err(err).Msg("Failed to close conntrack events")
		}
		close(done)
	}()
	for {
		select {
		case <-evChan:
		case <-errChan:
		case <-done:
			return
		}
	}
}

func isShutdownErr(ctx context.Context, c *Client, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errClientClosed) {
		return true
	}
	if ctx.Err() != nil || c.isClosing() {
		return true
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
