package issues

import (
	"hidevops.io/hiboot/pkg/app/web"
	"net/http"
	"testing"
)

func TestRun(t *testing.T) {
	web.RunTestApplication(t, new(TestController)).
		Get("/test").
		Expect().Status(http.StatusOK)
}
