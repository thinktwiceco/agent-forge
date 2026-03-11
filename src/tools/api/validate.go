package api

func (a *Api) getServiceNames() []string {
	names := make([]string, 0, len(a.services))
	for name := range a.services {
		names = append(names, name)
	}
	return names
}

func (a *Api) findService(name string) (ServiceConfig, bool) {
	svc, ok := a.services[name]
	return svc, ok
}

func (a *Api) findEndpoint(svc ServiceConfig, name string) *Endpoint {
	for i := range svc.Endpoints {
		if svc.Endpoints[i].Name == name {
			return &svc.Endpoints[i]
		}
	}
	return nil
}
