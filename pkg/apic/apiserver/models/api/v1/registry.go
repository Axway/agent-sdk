package v1

import "fmt"

var (
	scopeKindMap = map[GroupKind]string{}

	resourceMap = map[GroupKind]string{}

	gvkSet = map[GroupVersionKind]bool{}

	plurals = map[string]string{}
)

// RegisterGVK registers a GroupVersionKind with optional scope and mandatory resource
func RegisterGVK(gvk GroupVersionKind, scopeKind, resource string) {
	if gvk.Group == "" || gvk.Kind == "" || gvk.APIVersion == "" || resource == "" {
		panic(fmt.Sprintf("encountered empty fields while registering gvk to resource: %+v, %s", gvk, resource))
	}

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

	plurals[gvk.Kind] = resource
}

// GetScope return the scope of a Kind
func GetScope(gv GroupKind) (k string, ok bool) {
	k, ok = scopeKindMap[gv]
	return
}

// GetResource returns the resource name of a Kind
func GetResource(gv GroupKind) (r string, ok bool) {
	r, ok = resourceMap[gv]
	return
}

// GVKSet -
func GVKSet() []GroupVersionKind {
	res := make([]GroupVersionKind, len(gvkSet))

	i := 0
	for k := range gvkSet {
		res[i] = k
		i++
	}

	return res
}

// GetPluralFromKind
func GetPluralFromKind(kind string) (p string, ok bool) {
	p, ok = plurals[kind]
	return
}
