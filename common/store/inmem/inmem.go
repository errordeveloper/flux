package inmem

import (
	"fmt"
	"log"
	"sync"

	"golang.org/x/net/context"

	"github.com/weaveworks/flux/common/daemon"
	"github.com/weaveworks/flux/common/data"
	"github.com/weaveworks/flux/common/store"
)

func NewInMemStore() store.Store {
	return &inmem{
		services:   make(map[string]data.Service),
		groupSpecs: make(map[string]map[string]data.ContainerRule),
		instances:  make(map[string]map[string]data.Instance),
	}
}

type inmem struct {
	services     map[string]data.Service
	groupSpecs   map[string]map[string]data.ContainerRule
	instances    map[string]map[string]data.Instance
	watchersLock sync.Mutex
	watchers     []watcher
}

type watcher struct {
	ctx  context.Context
	ch   chan<- data.ServiceChange
	opts store.QueryServiceOptions
}

func (w watcher) Done() <-chan struct{} {
	if w.ctx == nil {
		return nil
	}
	return w.ctx.Done()
}

func (s *inmem) fireEvent(ev data.ServiceChange, optsFilter func(store.QueryServiceOptions) bool) {
	s.watchersLock.Lock()
	watchers := s.watchers
	s.watchersLock.Unlock()

	for _, watcher := range watchers {
		if optsFilter == nil || optsFilter(watcher.opts) {
			select {
			case watcher.ch <- ev:
			case <-watcher.Done():
			}
		}
	}
}

func (s *inmem) Ping() error {
	return nil
}

func (s *inmem) CheckRegisteredService(name string) error {
	if _, found := s.services[name]; !found {
		return fmt.Errorf(`Not found "%s"`, name)
	}
	return nil
}

func (s *inmem) AddService(name string, svc data.Service) error {
	s.services[name] = svc
	s.groupSpecs[name] = make(map[string]data.ContainerRule)
	s.instances[name] = make(map[string]data.Instance)

	s.fireEvent(data.ServiceChange{name, false}, nil)
	log.Printf("inmem: service %s updated in store", name)
	return nil
}

func (s *inmem) RemoveService(name string) error {
	delete(s.services, name)
	delete(s.groupSpecs, name)
	delete(s.instances, name)

	s.fireEvent(data.ServiceChange{name, true}, nil)
	log.Printf("inmem: service %s removed from store", name)
	return nil
}

func (s *inmem) RemoveAllServices() error {
	for name, _ := range s.services {
		s.RemoveService(name)
	}
	return nil
}

func (s *inmem) GetService(name string, opts store.QueryServiceOptions) (*store.ServiceInfo, error) {
	svc, found := s.services[name]
	if !found {
		return nil, fmt.Errorf(`Not found "%s"`, name)
	}

	return s.makeServiceInfo(name, svc, opts), nil
}

func (s *inmem) makeServiceInfo(name string, svc data.Service, opts store.QueryServiceOptions) *store.ServiceInfo {
	info := &store.ServiceInfo{
		Name:    name,
		Service: svc,
	}

	if opts.WithInstances {
		for n, i := range s.instances[info.Name] {
			info.Instances = append(info.Instances,
				store.InstanceInfo{Name: n, Instance: i})
		}
	}

	if opts.WithContainerRules {
		for n, g := range s.groupSpecs[info.Name] {
			info.ContainerRules = append(info.ContainerRules,
				store.ContainerRuleInfo{Name: n, ContainerRule: g})
		}
	}

	return info
}

func (s *inmem) GetAllServices(opts store.QueryServiceOptions) ([]*store.ServiceInfo, error) {
	var svcs []*store.ServiceInfo

	for name, svc := range s.services {
		svcs = append(svcs, s.makeServiceInfo(name, svc, opts))
	}

	return svcs, nil
}

func withRuleChanges(opts store.QueryServiceOptions) bool {
	return opts.WithContainerRules
}

func (s *inmem) SetContainerRule(serviceName string, groupName string, spec data.ContainerRule) error {
	groupSpecs, found := s.groupSpecs[serviceName]
	if !found {
		return fmt.Errorf(`Not found "%s"`, serviceName)
	}

	groupSpecs[groupName] = spec
	s.fireEvent(data.ServiceChange{serviceName, false}, withRuleChanges)
	return nil
}

func (s *inmem) RemoveContainerRule(serviceName string, groupName string) error {
	groupSpecs, found := s.groupSpecs[serviceName]
	if !found {
		return fmt.Errorf(`Not found "%s"`, serviceName)
	}

	delete(groupSpecs, groupName)
	s.fireEvent(data.ServiceChange{serviceName, false}, withRuleChanges)
	return nil
}

func withInstanceChanges(opts store.QueryServiceOptions) bool {
	return opts.WithInstances
}

func (s *inmem) AddInstance(serviceName string, instanceName string, inst data.Instance) error {
	s.instances[serviceName][instanceName] = inst
	s.fireEvent(data.ServiceChange{serviceName, false}, withInstanceChanges)
	return nil
}

func (s *inmem) RemoveInstance(serviceName string, instanceName string) error {
	if _, found := s.instances[serviceName][instanceName]; !found {
		return fmt.Errorf("service '%s' has no instance '%s'",
			serviceName, instanceName)
	}

	delete(s.instances[serviceName], instanceName)
	s.fireEvent(data.ServiceChange{serviceName, false}, withInstanceChanges)
	return nil
}

func (s *inmem) WatchServices(ctx context.Context, res chan<- data.ServiceChange, _ daemon.ErrorSink, opts store.QueryServiceOptions) {
	s.watchersLock.Lock()
	defer s.watchersLock.Unlock()
	w := watcher{ctx, res, opts}
	s.watchers = append(s.watchers, w)

	// discard the watcher upon cancellation
	go func() {
		<-w.Done()

		s.watchersLock.Lock()
		defer s.watchersLock.Unlock()
		for i, w := range s.watchers {
			if w.ch == res {
				// need to make a copy
				s.watchers = append(append([]watcher{}, s.watchers[:i]...), s.watchers[i+1:]...)
				break
			}
		}
	}()
}
