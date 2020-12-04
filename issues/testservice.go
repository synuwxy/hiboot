package issues

import "hidevops.io/hiboot/pkg/app"

type TestService struct {
}

func init() {
	app.Register(newTestService)
}

func newTestService() *TestService {
	return &TestService{}
}
