package v1

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/google/uuid"
)

func event(eType apiv1.EventType, ri *apiv1.ResourceInstance) *apiv1.Event {
	return &apiv1.Event{
		ID:   uuid.New().String(),
		Type: eType,
		Payload: apiv1.EventPayload{
			GroupKind:  ri.GroupKind,
			Scope:      ri.Metadata.Scope,
			Tags:       ri.Tags,
			Attributes: ri.Attributes,
			ID:         ri.Metadata.ID,
			Name:       ri.Name,
			References: ri.Metadata.References, // needed ?
		},
	}
}

type fakeGroup map[string]fakeVersion

type fakeVersion map[string]*fakeUnscoped

type fakeAttribute struct {
	key   string
	value string
}

type fakeUnscoped struct {
	fakeScoped
	scopedKinds map[string]fakeByScope
}

func notFound(name, kind string) NotFoundError {
	return NotFoundError{[]apiv1.Error{{
		Status: 404,
		Title:  "Not found error",
		Detail: fmt.Sprintf("Resource %s of kind %s not found.", name, kind),
	}}}
}

func notFoundInScope(name, kind, scopeName string) NotFoundError {
	return NotFoundError{[]apiv1.Error{{
		Status: 404,
		Title:  "Not found error",
		Detail: fmt.Sprintf("Resource %s of kind %s not found in scope %s.", name, kind, scopeName),
	}}}
}

type unknownScope NotFoundError

// Create -
func (us unknownScope) Create(ri *apiv1.ResourceInstance, opts ...CreateOption) (*apiv1.ResourceInstance, error) {
	return us.CreateCtx(context.Background(), ri)
}

// CreateCtx -
func (us unknownScope) CreateCtx(_ context.Context, _ *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	return nil, NotFoundError(us)
}

// Delete -
func (us unknownScope) Delete(ri *apiv1.ResourceInstance) error {
	return us.DeleteCtx(context.Background(), ri)
}

// DeleteCtx -
func (us unknownScope) DeleteCtx(ctx context.Context, _ *apiv1.ResourceInstance) error {
	return NotFoundError(us)
}

// Get -
func (us unknownScope) Get(name string) (*apiv1.ResourceInstance, error) {
	return us.GetCtx(context.Background(), name)
}

// GetCtx -
func (us unknownScope) GetCtx(_ context.Context, _ string) (*apiv1.ResourceInstance, error) {
	return nil, NotFoundError(us)
}

// List -
func (us unknownScope) List(options ...ListOptions) ([]*apiv1.ResourceInstance, error) {
	return us.ListCtx(context.Background(), options...)
}

// ListCtx -
func (us unknownScope) ListCtx(_ context.Context, _ ...ListOptions) ([]*apiv1.ResourceInstance, error) {
	return nil, NotFoundError(us)
}

// Update -
func (us unknownScope) Update(ri *apiv1.ResourceInstance, opts ...UpdateOption) (*apiv1.ResourceInstance, error) {
	return us.UpdateCtx(context.Background(), ri, opts...)
}

// UpdateCtx -
func (us unknownScope) UpdateCtx(_ context.Context, _ *apiv1.ResourceInstance, opts ...UpdateOption) (*apiv1.ResourceInstance, error) {
	return nil, NotFoundError(us)
}

type fakeByScope struct {
	apiv1.GroupVersionKind
	scopeKind apiv1.GroupKind
	fks       map[string]*fakeScoped
}

// Create -
func (fk fakeByScope) Create(ri *apiv1.ResourceInstance, opts ...CreateOption) (*apiv1.ResourceInstance, error) {
	return fk.CreateCtx(context.Background(), ri, opts...)
}

// CreateCtx -
func (fk fakeByScope) CreateCtx(c context.Context, ri *apiv1.ResourceInstance, opts ...CreateOption) (*apiv1.ResourceInstance, error) {
	newri := *ri

	newri.GroupVersionKind = fk.GroupVersionKind
	newri.ResourceMeta.Metadata.Scope.Kind = fk.scopeKind.Kind

	return fk.fks[ri.ResourceMeta.Metadata.Scope.Name].CreateCtx(c, &newri, opts...)
}

// Delete -
func (fk fakeByScope) Delete(ri *apiv1.ResourceInstance) error {
	return fk.DeleteCtx(context.Background(), ri)
}

// DeleteCtx -
func (fk fakeByScope) DeleteCtx(c context.Context, ri *apiv1.ResourceInstance) error {
	newri := *ri

	newri.GroupVersionKind = fk.GroupVersionKind
	newri.ResourceMeta.Metadata.Scope.Kind = fk.scopeKind.Kind

	return fk.fks[ri.ResourceMeta.Metadata.Scope.Name].DeleteCtx(c, &newri)
}

// Get -
func (fk fakeByScope) Get(ri string) (*apiv1.ResourceInstance, error) {
	return fk.GetCtx(context.Background(), ri)
}

// GetCtx -
func (fk fakeByScope) GetCtx(_ context.Context, name string) (*apiv1.ResourceInstance, error) {
	split := strings.SplitN(name, `/`, 2)

	if len(split) == 2 {
		return fk.fks[split[0]].Get(split[1])
	}

	return nil, notFound("", fk.scopeKind.Kind)
}

// List -
func (fk fakeByScope) List(ri ...ListOptions) ([]*apiv1.ResourceInstance, error) {
	return fk.ListCtx(context.Background(), ri...)
}

// ListCtx -
func (fk fakeByScope) ListCtx(_ context.Context, options ...ListOptions) ([]*apiv1.ResourceInstance, error) {
	// TODO should list work for unscoped
	return nil, notFound("", "")
}

// Update -
func (fk fakeByScope) Update(ri *apiv1.ResourceInstance, opts ...UpdateOption) (*apiv1.ResourceInstance, error) {
	return fk.UpdateCtx(context.Background(), ri, opts...)
}

// UpdateCtx -
func (fk fakeByScope) UpdateCtx(c context.Context, ri *apiv1.ResourceInstance, opts ...UpdateOption) (*apiv1.ResourceInstance, error) {
	newri := *ri

	newri.GroupVersionKind = fk.GroupVersionKind
	newri.ResourceMeta.Metadata.Scope.Kind = fk.scopeKind.Kind

	return fk.fks[ri.ResourceMeta.Metadata.Scope.Name].UpdateCtx(c, &newri, opts...)
	// TODO should work if ri has scope name

}

// WithScope -
func (fk fakeByScope) WithScope(name string) Scoped {
	if s, ok := fk.fks[name]; !ok {
		return unknownScope(notFound(name, fk.scopeKind.Kind))
	} else {
		return s
	}
}

// Create -
func (fk *fakeUnscoped) Create(ri *apiv1.ResourceInstance, opts ...CreateOption) (*apiv1.ResourceInstance, error) {
	return fk.CreateCtx(context.Background(), ri, opts...)
}

// CreateCtx -
func (fk *fakeUnscoped) CreateCtx(_ context.Context, ri *apiv1.ResourceInstance, opts ...CreateOption) (*apiv1.ResourceInstance, error) {
	fk.lock.Lock()
	defer fk.lock.Unlock()

	return fk.create(ri, opts...)
}

// Delete -
func (fk *fakeUnscoped) Delete(ri *apiv1.ResourceInstance) error {
	return fk.DeleteCtx(context.Background(), ri)
}

// DeleteCtx -
func (fk *fakeUnscoped) DeleteCtx(_ context.Context, ri *apiv1.ResourceInstance) error {
	if fk == nil {
		return notFound(ri.Metadata.Scope.Name, ri.Metadata.Scope.Kind)
	}

	fk.lock.Lock()
	defer fk.lock.Unlock()

	_, ok := fk.resources[ri.Name]
	if !ok {
		return notFound(ri.Name, fk.Kind)
	}

	for _, sk := range fk.scopedKinds {
		sk.fks[ri.Name].deleteAll()

		sk.fks[ri.Name] = nil
	}

	return fk.fakeScoped.delete(ri)
}

// Get -
func (fk *fakeUnscoped) Get(ri string) (*apiv1.ResourceInstance, error) {
	return fk.GetCtx(context.Background(), ri)
}

// GetCtx -
func (fk *fakeUnscoped) GetCtx(_ context.Context, name string) (*apiv1.ResourceInstance, error) {
	fk.lock.Lock()
	defer fk.lock.Unlock()

	return fk.get(name)
}

// WithScope -
func (fk *fakeScoped) WithScope(name string) Scoped {
	return (*fakeScoped)(nil)
}

func (fk *fakeUnscoped) create(ri *apiv1.ResourceInstance, opts ...CreateOption) (*apiv1.ResourceInstance, error) {
	created, err := fk.fakeScoped.create(ri, opts...)
	if err != nil {
		return created, err
	}

	for kind, scoped := range fk.scopedKinds {
		scoped.fks[created.Name] = newFakeKind(
			apiv1.GroupVersionKind{
				GroupKind: apiv1.GroupKind{
					Kind:  kind,
					Group: fk.Group,
				},
				APIVersion: fk.APIVersion,
			},
			apiv1.MetadataScope{
				ID:   created.Metadata.ID,
				Kind: fk.GroupVersionKind.Kind,
				Name: created.Name,
			},
			fk.handler,
		)
	}

	return created, nil
}

// WithScope -
func (fk *fakeUnscoped) WithScope(name string) Scoped {
	return (*fakeScoped)(nil)
}

type set map[string]struct{}

func newSet(entries ...string) set {
	s := set{}
	for _, entry := range entries {
		s[entry] = struct{}{}
	}
	return s
}

// Union -
func (s set) Union(other set) set {
	res := set{}
	for k, v := range s {
		res[k] = v
	}

	for k, v := range other {
		res[k] = v
	}

	return res
}

// Intersection -
func (s set) Intersection(other set) set {
	res := set{}
	for k, v := range s {
		if _, ok := other[k]; ok {
			res[k] = v
		}
	}

	return res
}

type index map[string][]string

// LookUp -
func (idx index) LookUp(key string) set {
	names, ok := idx[key]
	if !ok {
		return set{}
	}

	return newSet(names...)
}

// Update -
func (idx index) Update(old []string, new []string, val string) {
	toDelete := append([]string{}, old...)
	toAdd := append([]string{}, new...)

	n := 0
outer:
	for _, old := range toDelete {
		for j, new := range toAdd {
			if old == new {
				toAdd[j] = toAdd[len(toAdd)-1]
				toAdd = toAdd[:len(toAdd)-1]
				continue outer
			}
		}
		toDelete[n] = old
		n++
	}

	toDelete = toDelete[:n]

	for _, del := range toDelete {
		names, ok := idx[del]
		if !ok {
			panic(fmt.Sprintf("Trying to delete unknown index %s", del))
		}

		for i := range names {
			if names[i] == val {
				names[i] = names[len(names)-1]
				idx[del] = names[:len(names)-1]
				break
			}
		}
	}

	for _, add := range toAdd {
		names, ok := idx[add]
		if !ok {
			names = []string{}
		}
		idx[add] = append(names, val)
	}
}

type FakeVisitor struct {
	resources *fakeScoped
	set
}

// Visit -
func (fv *FakeVisitor) Visit(node QueryNode) {
	switch n := node.(type) {
	case andNode:
		for i, child := range n {
			childFV := &FakeVisitor{fv.resources, set{}}
			child.Accept(childFV)
			if i == 0 {
				fv.set = childFV.set
			} else {
				fv.set = fv.set.Intersection(childFV.set)
			}
		}
	case orNode:
		for _, child := range n {
			childFV := &FakeVisitor{fv.resources, set{}}
			child.Accept(childFV)
			fv.set = fv.set.Union(childFV.set)
		}
	case namesNode:
		fv.set = newSet(n...).Intersection(fv.resources.nameSet())
	case tagNode:
		for _, tag := range n {
			fv.set = fv.set.Union(fv.resources.tagsIndex.LookUp(tag))
		}
	case *attrNode:
		for _, val := range n.values {
			fv.set = fv.set.Union(fv.resources.attributeIndex.LookUp(fmt.Sprintf("%s;%s", n.key, val)))
		}
	default:
		panic(fmt.Sprintf("unknown node type %+v", n))
	}
}

func attrsAsIdxs(attrs map[string]string) []string {
	// update attributes
	idxs := make([]string, len(attrs))

	for key, val := range attrs {
		idxs = append(idxs, fmt.Sprintf("%s;%s", key, val))
	}
	return idxs
}

type fakeScoped struct {
	apiv1.GroupVersionKind
	ms             apiv1.MetadataScope
	resources      map[string]*apiv1.ResourceInstance
	tagsIndex      index
	attributeIndex index
	lock           *sync.Mutex
	handler        EventHandler
}

func (fk *fakeScoped) nameSet() set {
	res := make(set, len(fk.resources))
	for k := range fk.resources {
		res[k] = struct{}{}
	}

	return res
}

// Create -
func (fk *fakeScoped) Create(ri *apiv1.ResourceInstance, opts ...CreateOption) (*apiv1.ResourceInstance, error) {
	return fk.CreateCtx(context.Background(), ri, opts...)
}

// CreateCtx -
func (fk *fakeScoped) CreateCtx(_ context.Context, ri *apiv1.ResourceInstance, opts ...CreateOption) (*apiv1.ResourceInstance, error) {
	if fk == nil {
		return nil, notFound(ri.Metadata.Scope.Name, ri.Metadata.Scope.Kind)
	}

	fk.lock.Lock()
	defer fk.lock.Unlock()

	return fk.create(ri, opts...)
}

// Delete -
func (fk *fakeScoped) Delete(ri *apiv1.ResourceInstance) error {
	return fk.DeleteCtx(context.Background(), ri)
}

// DeleteCtx -
func (fk *fakeScoped) DeleteCtx(_ context.Context, ri *apiv1.ResourceInstance) error {
	if fk == nil {
		return notFound(ri.Metadata.Scope.Name, ri.Metadata.Scope.Kind)
	}

	fk.lock.Lock()
	defer fk.lock.Unlock()

	return fk.delete(ri)
}

func (fk *fakeScoped) delete(ri *apiv1.ResourceInstance) error {
	deleted, ok := fk.resources[ri.Name]
	if !ok {
		return notFoundInScope(ri.Name, fk.Kind, fk.ms.Name)
	}

	fk.attributeIndex.Update(attrsAsIdxs(deleted.Attributes), []string{}, deleted.Name)
	fk.tagsIndex.Update(deleted.Tags, []string{}, deleted.Name)

	fk.handler.Handle(event(apiv1.ResourceEntryDeletedEvent, deleted))

	return nil
}

// Get -
func (fk *fakeScoped) Get(ri string) (*apiv1.ResourceInstance, error) {
	return fk.GetCtx(context.Background(), ri)
}

// GetCtx -
func (fk *fakeScoped) GetCtx(_ context.Context, name string) (*apiv1.ResourceInstance, error) {
	fk.lock.Lock()
	defer fk.lock.Unlock()

	return fk.get(name)
}

// List -
func (fk *fakeScoped) List(ri ...ListOptions) ([]*apiv1.ResourceInstance, error) {
	return fk.ListCtx(context.Background(), ri...)
}

// ListCtx -
func (fk *fakeScoped) ListCtx(_ context.Context, options ...ListOptions) ([]*apiv1.ResourceInstance, error) {
	if fk == nil {
		return nil, fmt.Errorf("unknown scope") // TODO
	}

	opts := listOptions{}

	for _, o := range options {
		o(&opts)
	}

	fk.lock.Lock()
	defer fk.lock.Unlock()

	if opts.query == nil {
		ris := make([]*apiv1.ResourceInstance, len(fk.resources))

		i := 0
		for _, ri := range fk.resources {
			ris[i] = ri
			i++
		}
		return ris, nil
	}

	fv := &FakeVisitor{
		resources: fk,
		set:       set{},
	}

	opts.query.Accept(fv)

	ris := make([]*apiv1.ResourceInstance, len(fv.set))

	i := 0
	for k := range fv.set {
		if ri, ok := fk.resources[k]; !ok {
			panic(fmt.Sprintf("Resource %s in index but not in resource list", k))
		} else {
			ris[i] = ri
			i++
		}
	}

	return ris, nil
}

// Update -
func (fk *fakeScoped) Update(ri *apiv1.ResourceInstance, opts ...UpdateOption) (*apiv1.ResourceInstance, error) {
	return fk.UpdateCtx(context.Background(), ri, opts...)
}

// UpdateCtx -
func (fk *fakeScoped) UpdateCtx(_ context.Context, ri *apiv1.ResourceInstance, opts ...UpdateOption) (*apiv1.ResourceInstance, error) {
	if fk == nil {
		return nil, notFound(ri.Metadata.Scope.Name, ri.Metadata.Scope.Kind)
	}

	fk.lock.Lock()
	defer fk.lock.Unlock()

	return fk.update(ri, opts...)
}

func (fk *fakeScoped) create(ri *apiv1.ResourceInstance, opts ...CreateOption) (*apiv1.ResourceInstance, error) {

	co := createOptions{}

	for _, opt := range opts {
		opt(&co)
	}

	if ri.Name == "" {
		return nil, fmt.Errorf("empty resource name: %v", ri)
	}

	if ex, ok := fk.resources[ri.Name]; ok {
		return nil, fmt.Errorf("existing resource: %v", ex)
	}

	created := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Name:             ri.Name,
			Title:            ri.Title,
			GroupVersionKind: fk.GroupVersionKind,
			Metadata: apiv1.Metadata{
				ID: uuid.New().String(),
				Audit: apiv1.AuditMetadata{
					CreateTimestamp: apiv1.Time(time.Now()),
					CreateUserID:    co.impersonateUserID,
					ModifyTimestamp: apiv1.Time(time.Now()),
					ModifyUserID:    co.impersonateUserID,
				},
				Scope:           fk.ms,
				ResourceVersion: "0",
				References:      nil,
				State:           "", // TODO
			},
			Attributes: ri.Attributes,
			Tags:       ri.Tags,
		},
		Spec: ri.Spec,
	}

	fk.attributeIndex.Update([]string{}, attrsAsIdxs(created.Attributes), created.Name)
	fk.tagsIndex.Update([]string{}, created.Tags, created.Name)

	fk.resources[ri.Name] = created

	fk.handler.Handle(event(apiv1.ResourceEntryCreatedEvent, created))

	return created, nil
}

func (fk *fakeScoped) update(ri *apiv1.ResourceInstance, opts ...UpdateOption) (*apiv1.ResourceInstance, error) {
	uo := updateOptions{}

	for _, opt := range opts {
		opt(&uo)
	}

	if ri.Name == "" {
		return nil, notFound(ri.Metadata.Scope.Name, ri.Metadata.Scope.Kind)
	}

	prev, ok := fk.resources[ri.Name]
	if !ok && uo.mergeFunc == nil {
		return nil, notFoundInScope(ri.Name, fk.Kind, fk.ms.Name)
	}

	if uo.mergeFunc != nil {
		merged, err := uo.mergeFunc(prev, ri)
		if err != nil {
			return nil, err
		}

		ri, err = merged.AsInstance()
		if err != nil {
			return nil, err
		}

		if !ok {
			return fk.create(ri, CUserID(uo.impersonateUserID))
		}
	}

	updated := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Name:             prev.Name,
			Title:            prev.Title,
			GroupVersionKind: prev.GroupVersionKind,
			Metadata: apiv1.Metadata{
				ID: prev.Metadata.ID,
				Audit: apiv1.AuditMetadata{
					CreateTimestamp: prev.Metadata.Audit.CreateTimestamp,
					CreateUserID:    prev.Metadata.Audit.CreateUserID,
					ModifyTimestamp: apiv1.Time(time.Now()),
					ModifyUserID:    uo.impersonateUserID,
				},
				Scope:           prev.Metadata.Scope,
				ResourceVersion: prev.Metadata.ResourceVersion,
				References:      nil,
				State:           "", // needed?
			},
			Attributes: ri.Attributes,
			Tags:       ri.Tags,
		},
		Spec: ri.Spec,
	}

	// update indexes
	fk.attributeIndex.Update(attrsAsIdxs(prev.Attributes), attrsAsIdxs(updated.Attributes), updated.Name)
	fk.tagsIndex.Update(prev.Tags, updated.Tags, updated.Name)

	fk.resources[ri.Name] = updated

	fk.handler.Handle(event(apiv1.ResourceEntryUpdatedEvent, updated))

	return updated, nil
}

func (fk *fakeScoped) get(name string) (*apiv1.ResourceInstance, error) {
	if fk == nil {
		return nil, &NotFoundError{[]apiv1.Error{}}
	}

	ris, ok := fk.resources[name]
	if !ok {
		return nil, notFoundInScope(name, fk.Kind, fk.ms.Name)
	}

	return ris, nil
}

func (fk *fakeScoped) deleteAll() error {
	fk.lock.Lock()
	defer fk.lock.Unlock()

	for _, ri := range fk.resources {
		fk.handler.Handle(event(apiv1.ResourceEntryDeletedEvent, ri))
	}

	*fk = fakeScoped{}

	return nil
}

type delegatingEventHandler struct {
	wrapped EventHandler
}

// Handle -
func (dh *delegatingEventHandler) Handle(e *apiv1.Event) {
	if dh != nil && dh.wrapped != nil {
		go func() {
			if dh != nil && dh.wrapped != nil {
				dh.wrapped.Handle(e)
			}
		}()
	}
}

type fakeClientBase struct {
	handler *delegatingEventHandler
	groups  map[string]fakeGroup
}

type fakeClient struct {
	fakeClientBase
	Unscoped
}

func newFakeKind(gvk apiv1.GroupVersionKind, ms apiv1.MetadataScope, handler EventHandler) *fakeScoped {
	return &fakeScoped{
		gvk,
		ms,
		map[string]*apiv1.ResourceInstance{},
		index{},
		index{},
		&sync.Mutex{},
		handler,
	}
}

// NewFakeClient -
func NewFakeClient(is ...apiv1.Interface) (*fakeClientBase, error) {
	handler := &delegatingEventHandler{}
	groups := map[string]fakeGroup{}

	for _, gvk := range apiv1.GVKSet() {

		group, ok := groups[gvk.Group]
		if !ok {
			group = map[string]fakeVersion{}
			groups[gvk.Group] = group
		}

		version, ok := group[gvk.APIVersion]
		if !ok {
			version = fakeVersion(map[string]*fakeUnscoped{})
			group[gvk.APIVersion] = version
		}

		sk, ok := apiv1.GetScope(gvk.GroupKind)
		if !ok {
			panic(fmt.Sprintf("no scope for gvk: %s", gvk))
		}

		if sk != "" {
			scope, ok := version[sk]
			if !ok {
				scope = &fakeUnscoped{
					*newFakeKind(
						apiv1.GroupVersionKind{
							GroupKind: apiv1.GroupKind{
								Group: gvk.Group,
								Kind:  sk,
							},
							APIVersion: gvk.APIVersion,
						},
						apiv1.MetadataScope{},
						handler,
					),
					map[string]fakeByScope{},
				}
				version[sk] = scope
			}

			_, ok = scope.scopedKinds[gvk.Kind]
			if !ok {
				scope.scopedKinds[gvk.Kind] = fakeByScope{
					GroupVersionKind: gvk,
					scopeKind:        scope.GroupKind,
					fks:              map[string]*fakeScoped{},
				}
			}

			continue
		}

		if _, ok := version[gvk.Kind]; !ok {
			version[gvk.Kind] = &fakeUnscoped{
				*newFakeKind(
					gvk,

					apiv1.MetadataScope{},
					handler,
				),
				map[string]fakeByScope{},
			}
		}
	}

	client := &fakeClientBase{handler, groups}

	// pass through and create unscoped resources
	for _, i := range is {
		ri, err := i.AsInstance()
		if err != nil {
			return nil, err
		}

		sk, ok := apiv1.GetScope(ri.GroupKind)
		if !ok {
			return nil, fmt.Errorf("no scope kind or unknown kind for ri: %v", ri)
		}
		if sk != "" {
			continue
		}

		c, err := client.ForKind(ri.GroupVersionKind)
		if err != nil {
			return nil, err
		}

		_, err = c.Create(ri)
		if err != nil {
			return nil, err
		}
	}

	// pass through and create scoped resources
	for _, i := range is {
		ri, err := i.AsInstance()
		if err != nil {
			return nil, err
		}

		sk, ok := apiv1.GetScope(ri.GroupKind)
		if !ok {
			return nil, fmt.Errorf("no scope kind or unknown kind for ri: %v", ri)
		}
		if sk == "" {
			continue
		}

		noScope, err := client.ForKind(ri.GroupVersionKind)
		if err != nil {
			return nil, err
		}

		c := noScope.WithScope(ri.Metadata.Scope.Name)

		_, err = c.Create(ri)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

// ForKind -
func (fcb fakeClientBase) ForKind(gvk apiv1.GroupVersionKind) (Unscoped, error) {
	sk, ok := apiv1.GetScope(gvk.GroupKind)
	if !ok {
		panic(fmt.Sprintf("no scope for gvk: %s", gvk))
	}

	if sk == "" {
		return fcb.groups[gvk.Group][gvk.APIVersion][gvk.Kind], nil
	}

	return fcb.groups[gvk.Group][gvk.APIVersion][sk].scopedKinds[gvk.Kind], nil
}

// SetHandler -
func (fcb fakeClientBase) SetHandler(ev EventHandler) {
	fcb.handler.wrapped = ev
}
