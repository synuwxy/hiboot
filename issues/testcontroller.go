package issues

import (
	"hidevops.io/hiboot/pkg/app"
	"hidevops.io/hiboot/pkg/at"
	"hidevops.io/hiboot/pkg/model"
)

type TestController struct {
	at.RestController
	at.RequestMapping `value:"/test"`
	testNewService    TestService
}

func init() {
	app.Register(newTestController)
}

func newTestController(testService TestService) *TestController {
	return &TestController{
		testNewService: testService,
	}
}

func (tc *TestController) Get(
	_ struct {
		at.GetMapping `value:"/"`
		Request       struct {
		}
		Response struct {
			StatusOK struct {
				at.Response `code:"200" description:"get agent state success"`
			}
		}
	}) (response *model.BaseResponseInfo, err error) {
	response = new(model.BaseResponseInfo)
	return response, nil
}
