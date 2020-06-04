package v1

import "fmt"

var (
	scopeKindMap = map[GroupKind]string{}

	resourceMap = map[GroupKind]string{}

	gvkSet = map[GroupVersionKind]bool{}
)

// RegisterGVK registers a GroupVersionKind with optional scope and mandatory resource
func RegisterGVK(gvk GroupVersionKind, scopeKind string, resource string) {
	// TODO gvk must not have empty fields

	// TODO Resource must not be be empty
	if gvkSet[gvk] {
		panic(fmt.Sprint("Attempt to register duplicate gvk: ", gvk))
	}
	gvkSet[gvk] = true

	if sk, ok := scopeKindMap[gvk.GroupKind]; ok && sk != scopeKind {
		panic(fmt.Sprintf("Attempt to set different scope: %s for gvk: %v. Previously set scope: %s", sk, gvk, sk))
	}

	scopeKindMap[gvk.GroupKind] = scopeKind

	if r, ok := resourceMap[gvk.GroupKind]; ok && r != resource {
		panic(fmt.Sprintf("Attempt to register different resurce: %s for gvk: %v. Previously set resource: %s", scopeKind, gvk, r))
	}

	resourceMap[gvk.GroupKind] = resource
}

func GetScope(gv GroupKind) (k string, ok bool) {
	k, ok = scopeKindMap[gv]
	return
}

func GetResource(gv GroupKind) (r string, ok bool) {
	r, ok = resourceMap[gv]
	return
}
