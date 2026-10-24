package span

import (
	"context"
	"fmt"

	"github.com/thread_koder/mochi/core/internal/database"
)

type PodIdentity struct {
	UID       string
	Namespace string
	Name      string
}

// PodIdentityCache is pass-local so one OTLP batch does not re-hit the same pod.
type PodIdentityCache struct {
	byUID  map[string]PodIdentity
	byName map[string]PodIdentity
}

func NewPodIdentityCache() *PodIdentityCache {
	return &PodIdentityCache{
		byUID:  make(map[string]PodIdentity),
		byName: make(map[string]PodIdentity),
	}
}

func nameKey(namespace, name string) string {
	return namespace + "/" + name
}

func FillPodIdentity(ctx context.Context, cache *PodIdentityCache, uid, namespace, name string) (PodIdentity, error) {
	if uid != "" {
		if hit, ok := cache.byUID[uid]; ok {
			return mergeIdentity(hit, namespace, name), nil
		}
		pod, err := database.GetPodIdentityByUID(ctx, uid)
		if err != nil {
			return PodIdentity{}, fmt.Errorf("lookup pod uid %s: %w", uid, err)
		}
		id := PodIdentity{UID: pod.UID, Namespace: pod.Namespace, Name: pod.Name}
		cache.byUID[uid] = id
		if id.Namespace != "" && id.Name != "" {
			cache.byName[nameKey(id.Namespace, id.Name)] = id
		}
		return mergeIdentity(id, namespace, name), nil
	}
	if namespace == "" || name == "" {
		return PodIdentity{}, fmt.Errorf("missing k8s.pod.uid and k8s.pod.name/namespace")
	}
	key := nameKey(namespace, name)
	if hit, ok := cache.byName[key]; ok {
		return hit, nil
	}
	pod, err := database.GetPodByName(ctx, name, namespace)
	if err != nil {
		return PodIdentity{}, fmt.Errorf("lookup pod %s/%s: %w", namespace, name, err)
	}
	id := PodIdentity{UID: pod.UID, Namespace: pod.Namespace, Name: pod.Name}
	cache.byName[key] = id
	cache.byUID[id.UID] = id
	return id, nil
}

func mergeIdentity(id PodIdentity, namespace, name string) PodIdentity {
	if id.Namespace == "" {
		id.Namespace = namespace
	}
	if id.Name == "" {
		id.Name = name
	}
	return id
}
