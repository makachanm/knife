package base

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"slices"
	"strings"
)

type APIContext struct {
	req http.Request
	res http.ResponseWriter

	httpType    string
	middleWares []APIMiddleware

	rawBodyData []byte
}

type APIError struct {
	ErrorCode *int   `json:"-"`
	ErrorType string `json:"error"`
	ErrorDesc string `json:"description"`
}

func NewAPIError(errortype string, errordesc string, errorcode int) APIError {
	return APIError{
		ErrorType: errortype,
		ErrorDesc: errordesc,
		ErrorCode: &errorcode,
	}
}

type APIMiddleware interface {
	RunMiddleware(rs http.ResponseWriter, rq *http.Request) APIMiddlewareResult
	GetMiddlewareInfo() APIMiddlewareInfo
}

type APIMiddlewareInfo struct {
	MiddlewareName string
}

type APIMiddlewareResult struct {
	IsSuccess bool
	ApiError  APIError
}

func (context *APIContext) runMiddlewars(allowed []string) bool {
	if len(allowed) <= 0 {
		return true
	}

	if allowed[0] == "all" {
		for _, middleware := range context.middleWares {
			result := middleware.RunMiddleware(context.res, &context.req)
			if !result.IsSuccess {
				context.ReturnError(result.ApiError.ErrorType, result.ApiError.ErrorDesc, *result.ApiError.ErrorCode)
				return false
			}
		}
	} else {
		for _, middleware := range context.middleWares {
			mwname := middleware.GetMiddlewareInfo().MiddlewareName
			if !slices.Contains(allowed, mwname) {
				continue
			}

			result := middleware.RunMiddleware(context.res, &context.req)
			if !result.IsSuccess {
				context.ReturnError(result.ApiError.ErrorType, result.ApiError.ErrorDesc, *result.ApiError.ErrorCode)
				return false
			}
		}
	}

	return true
}

func (context *APIContext) GetCookie(name string) (*http.Cookie, error) {
	return context.req.Cookie(name)
}

func (context *APIContext) GetPathParamValue(key string) string {
	return context.req.PathValue(key)
}

func (context *APIContext) GetHost() string {
	return context.req.Host
}

func (context *APIContext) GetRequest() *http.Request {
	return &context.req
}

func (context *APIContext) SetHeader(key, value string) {
	context.res.Header().Set(key, value)
}

func (context *APIContext) GetContext(v any) error {
	q := context.req.URL.Query()
	entity := reflect.ValueOf(v).Elem()

	for i := 0; i < entity.NumField(); i++ {
		fieldname, fieldexist := entity.Type().Field(i).Tag.Lookup("param")
		if q.Has(fieldname) && fieldexist {
			entity.Field(i).SetString(q.Get(fieldname))
		} else {
			entity.Field(i).SetZero()
		}
	}

	if context.httpType == http.MethodPost {
		//bodydata, _ := io.ReadAll(context.req.Body)

		err := json.Unmarshal(context.rawBodyData, &v)
		if err != nil {
			log.Fatalf(err.Error())
			context.ReturnError("servererror", "internal server deserialize error", http.StatusInternalServerError)
			return err
		}
	}

	return nil
}

func (context *APIContext) ReturnJSON(v any) error {
	marshalled, err := json.Marshal(v)

	if err != nil {
		/*
			context.res.WriteHeader(http.StatusInternalServerError)
			errorstr := `{"servererror":"failed to pack a data"}`
			context.res.Write([]byte(errorstr))
		*/
		context.ReturnError("servererror", "failed to pack a data", http.StatusInternalServerError)
		return err
	}

	context.res.WriteHeader(http.StatusOK)
	context.res.Header().Set("Content-Type", "application/json")
	context.res.Write(marshalled)
	return nil
}

func (context *APIContext) RawRetrun(data []byte, returncode int) error {
	context.res.WriteHeader(returncode)
	_, err := context.res.Write(data)
	return err
}

func (context *APIContext) RawBody() []byte {
	//data, err := io.ReadAll(context.req.Body)
	//if err != nil {
	//	context.ReturnError("servererror", "internal server decode error", http.StatusInternalServerError)
	//	return []byte{}
	//}
	return context.rawBodyData
}

func (context *APIContext) ReturnError(errortype string, errordescript string, returncode int) error {
	context.res.WriteHeader(returncode)
	ax := APIError{
		ErrorType: errortype,
		ErrorDesc: errordescript,
	}

	return context.ReturnJSON(ax)
}

func (ctx *APIContext) SetCookie(name, value, path string, maxAge int64, httpOnly bool) {
	http.SetCookie(ctx.res, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   int(maxAge),
		HttpOnly: httpOnly,
		SameSite: http.SameSiteStrictMode, // 쿠키가 동일 사이트에서만 전송되도록 설정
	})
}

type APIRouter struct {
	mux http.ServeMux

	prefix      string
	middlewares []APIMiddleware
}

func NewAPIRouter() APIRouter {
	return APIRouter{
		mux:         *http.NewServeMux(),
		prefix:      "",
		middlewares: make([]APIMiddleware, 0),
	}
}

func (router *APIRouter) RegisterMidddleware(mw APIMiddleware) {
	router.middlewares = append(router.middlewares, mw)
}

func (router *APIRouter) SetPrefix(prefix string) {
	router.prefix = prefix
}

func (router *APIRouter) pathMaker(path string, method string) string {
	var endpath string = ""

	if router.prefix != "" {
		endpath = strings.Join([]string{method, " ", "/", router.prefix, "/", path}, "")
	} else {
		endpath = strings.Join([]string{method, " ", "/", path}, "")
	}

	fmt.Println(endpath)

	return endpath
}

func (router *APIRouter) GET(path string, delegate func(APIContext), allowMiddleware []string) {
	interceptor := func(resx http.ResponseWriter, reqx *http.Request) {
		bodydata, berr := io.ReadAll(reqx.Body)

		if berr != nil {
			rctx := APIContext{res: resx, req: *reqx, httpType: reqx.Method}
			rctx.ReturnError("servererror", "internal bytestream read error", http.StatusInternalServerError)
		}

		ctx := APIContext{
			res: resx,
			req: *reqx.Clone(context.Background()),

			httpType:    reqx.Method,
			middleWares: router.middlewares,

			rawBodyData: bodydata,
		}

		success := ctx.runMiddlewars(allowMiddleware)
		if !success {
			return
		}
		delegate(ctx)
	}
	router.mux.HandleFunc(router.pathMaker(path, "GET"), interceptor)
}

func (router *APIRouter) POST(path string, delegate func(APIContext), allowMiddleware []string) {
	interceptor := func(resx http.ResponseWriter, reqx *http.Request) {
		bodydata, berr := io.ReadAll(reqx.Body)

		if berr != nil {
			rctx := APIContext{res: resx, req: *reqx, httpType: reqx.Method}
			rctx.ReturnError("servererror", "internal bytestream read error", http.StatusInternalServerError)
		}

		ctx := APIContext{
			res: resx,
			req: *reqx,

			httpType:    reqx.Method,
			middleWares: router.middlewares,

			rawBodyData: bodydata,
		}

		success := ctx.runMiddlewars(allowMiddleware)
		if !success {
			return
		}
		delegate(ctx)
	}
	router.mux.HandleFunc(router.pathMaker(path, "POST"), interceptor)
}

func (router *APIRouter) PUT(path string, delegate func(APIContext), allowMiddleware []string) {
	interceptor := func(resx http.ResponseWriter, reqx *http.Request) {
		bodydata, berr := io.ReadAll(reqx.Body)

		if berr != nil {
			rctx := APIContext{res: resx, req: *reqx, httpType: reqx.Method}
			rctx.ReturnError("servererror", "internal bytestream read error", http.StatusInternalServerError)
		}

		ctx := APIContext{
			res: resx,
			req: *reqx,

			httpType:    reqx.Method,
			middleWares: router.middlewares,

			rawBodyData: bodydata,
		}

		success := ctx.runMiddlewars(allowMiddleware)
		if !success {
			return
		}
		delegate(ctx)
	}
	router.mux.HandleFunc(router.pathMaker(path, "PUT"), interceptor)
}

func (router *APIRouter) DELETE(path string, delegate func(APIContext), allowMiddleware []string) {
	interceptor := func(resx http.ResponseWriter, reqx *http.Request) {
		bodydata, berr := io.ReadAll(reqx.Body)

		if berr != nil {
			rctx := APIContext{res: resx, req: *reqx, httpType: reqx.Method}
			rctx.ReturnError("servererror", "internal bytestream read error", http.StatusInternalServerError)
		}

		ctx := APIContext{
			res: resx,
			req: *reqx.Clone(context.Background()),

			httpType:    reqx.Method,
			middleWares: router.middlewares,

			rawBodyData: bodydata,
		}

		success := ctx.runMiddlewars(allowMiddleware)
		if !success {
			return
		}
		delegate(ctx)
	}
	router.mux.HandleFunc(router.pathMaker(path, "DELETE"), interceptor)
}

func (router *APIRouter) GetMUX() *http.ServeMux {
	return &router.mux
}
