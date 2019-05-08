package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
)

const (
	startParam = 14
)

type Params struct {
	URL string `json:"url"`
	Method string `json:"method,omitempty"`
	Auth bool `json:"auth"`
}

type Function struct {
	paramsStruct string
	params Params
	name string
}

type StructParams struct {
	tag reflect.StructTag
	paramType string
	name string
}

var (
	functions = make(map[string][]Function)
	structParams = make(map[string][]StructParams)
)

func main() {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if nil != err {
		panic(err)
	}

	out, err := os.Create(os.Args[2])
	if nil != err {
		panic(err)
	}
	defer func() {
		err := out.Close()
		if nil != err {
			panic(err)
		}
	}()

	fmt.Fprintln(out, `package ` + file.Name.Name)
	fmt.Fprintln(out, `import "net/http"`)
	fmt.Fprintln(out, `import "strconv"`)
	fmt.Fprintln(out, `import "context"`)
	fmt.Fprintln(out, `import "encoding/json"`)
	fmt.Fprintln(out, `import "errors"`)
	fmt.Fprintln(out)

	params := Params{}
	var paramsBin bytes.Buffer
	var takeFunc bool
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if nil == funcDecl.Doc {
			continue
		}

		for _, doc := range funcDecl.Doc.List {
			if !strings.HasPrefix(doc.Text, "// apigen:api {") {
				continue
			}

			paramsBin = bytes.Buffer{}
			paramsBin.WriteString(doc.Text[startParam:])

			err = json.Unmarshal(paramsBin.Bytes(), &params)
			if nil != err {
				panic(err)
			}

			takeFunc = true
			break
		}

		if !takeFunc {
			continue
		}

		recv := funcDecl.Recv
		if nil == recv {
			continue
		}

		list := recv.List
		if nil == list {
			continue
		}

		starExpr, ok := list[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}

		ident, ok := starExpr.X.(*ast.Ident)
		if !ok {
			continue
		}

		baseStruct := ident.Name

		funcType := funcDecl.Type
		if nil == funcType {
			continue
		}

		fieldList := funcType.Params
		if nil == fieldList {
			continue
		}

		list = fieldList.List
		if len(list) != 2 {
			continue
		}

		ident, ok = list[1].Type.(*ast.Ident)
		if !ok {
			continue
		}

		paramsStruct := ident.Name

		functions[baseStruct] = append(functions[baseStruct], Function{params: params, paramsStruct: paramsStruct, name: funcDecl.Name.Name})

		takeFunc = false
	}

	for _, decls := range file.Decls {
		gen, ok := decls.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			fieldList := structType.Fields
			if nil == fieldList {
				continue
			}

			for _, field := range fieldList.List {
				if nil == field.Tag  {
					continue
				}

				structParams[typeSpec.Name.Name] = append(structParams[typeSpec.Name.Name], StructParams{
					paramType: field.Type.(*ast.Ident).Name,
					tag: reflect.StructTag(field.Tag.Value[1:len(field.Tag.Value) - 1]),
					name: field.Names[0].Name,
				})
			}
		}
	}

	for baseStruct, structFunctions := range functions {
		fmt.Fprintln(out, "func (in *" + baseStruct + ") ServeHTTP(w http.ResponseWriter, r *http.Request) {")
		fmt.Fprintln(out,"\tswitch r.URL.Path {")
		methods := make(map[string][]Function)
		for _, function := range structFunctions {
			methods[function.params.URL] = append(methods[function.params.URL], function)
		}
		for _, method := range methods {
			for _, function := range method {
				fmt.Fprintln(out, "\tcase \""+function.params.URL+"\":")
				switch function.params.Method {
				case "POST":
					fmt.Fprintln(out, "\t\tif r.Method == \"POST\" {")
					fmt.Fprintln(out, "\t\t\tin.handler"+function.name+"(w,r)")
					fmt.Fprintln(out, "\t\t\treturn")
					fmt.Fprintln(out, "\t\t}")
				case "GET":
					fmt.Fprintln(out, "\t\tif r.Method == \"GET\" {")
					fmt.Fprintln(out, "\t\t\tin.handler"+function.name+"(w,r)")
					fmt.Fprintln(out, "\t\t\treturn")
					fmt.Fprintln(out, "\t\t}")
				default:
					fmt.Fprintln(out, "\t\tif r.Method == \"GET\" {")
					fmt.Fprintln(out, "\t\t\tin.handler"+function.name+"(w,r)")
					fmt.Fprintln(out, "\t\t\treturn")
					fmt.Fprintln(out, "\t\t}")
					fmt.Fprintln(out, "\t\tif r.Method == \"POST\" {")
					fmt.Fprintln(out, "\t\t\tin.handler"+function.name+"(w,r)")
					fmt.Fprintln(out, "\t\t\treturn")
					fmt.Fprintln(out, "\t\t}")
				}
			}
			fmt.Fprintln(out, "\t\tapiError := ApiError{Err: errors.New(\"bad method\"), HTTPStatus: http.StatusNotAcceptable}")
			fmt.Fprintln(out, "\t\thandleError(w, apiError)")
			fmt.Fprintln(out, "\t\treturn")
		}
		fmt.Fprintln(out, "\t}")
		fmt.Fprintln(out, "\tapiError := ApiError{Err: errors.New(\"unknown method\"), HTTPStatus: http.StatusNotFound}")
		fmt.Fprintln(out, "\thandleError(w, apiError)")
		fmt.Fprintln(out, "}")
		fmt.Fprintln(out)

		for _, function := range structFunctions {
			fmt.Fprintln(out, "func (in *" + baseStruct + ") handler" + function.name + "(w http.ResponseWriter, r *http.Request) {")
			if function.params.Auth {
				fmt.Fprintln(out, "\ttoken := r.Header.Get(\"X-Auth\")")
				fmt.Fprintln(out, "\tif token != \"100500\" {")
				fmt.Fprintln(out, "\t\tapiError := ApiError{Err: errors.New(\"unauthorized\"), HTTPStatus: http.StatusForbidden}")
				fmt.Fprintln(out, "\t\thandleError(w, apiError)")
				fmt.Fprintln(out, "\t\treturn")
				fmt.Fprintln(out, "\t}")
			}
			fmt.Fprintln(out, "\tparams := " + function.paramsStruct + "{}")
			for _, structParam := range structParams[function.paramsStruct] {
				tags := structParam.tag.Get("apivalidator")
				if tags == "" {
					continue
				}

				tags = strings.Replace(tags, " ", "", -1)

				paramname := strings.ToLower(structParam.name)
				tagsArray := strings.Split(tags, ",")
				for _, tagExpr := range tagsArray {
					tagArray := strings.Split(tagExpr, "=")
					if tagArray[0] == "paramname" {
						paramname = tagArray[1]
						break
					}
				}
				switch structParam.paramType {
				case "int":
					fmt.Fprintln(out, "\t" + structParam.name + ", err := strconv.Atoi(r.FormValue(\"" + paramname + "\"))")
					fmt.Fprintln(out, "\tif nil != err {")
					fmt.Fprintln(out, "\t\tapiError := ApiError{Err: errors.New(\"" + paramname + " must be int\"), HTTPStatus: http.StatusBadRequest}")
					fmt.Fprintln(out, "\t\thandleError(w, apiError)")
					fmt.Fprintln(out, "\t\treturn")
					fmt.Fprintln(out, "\t}")
					fmt.Fprintln(out, "\tparams." + structParam.name + " = " + structParam.name)
				case "string":
					fmt.Fprintln(out, "\tparams." + structParam.name + " = r.FormValue(\""+paramname+"\")")
				}
			}
			for _, structParam := range structParams[function.paramsStruct] {
				tags := structParam.tag.Get("apivalidator")
				if tags == "" {
					continue
				}

				tags = strings.Replace(tags, " ", "", -1)

				paramname := strings.ToLower(structParam.name)
				tagsArray := strings.Split(tags, ",")
				for _, tagExpr := range tagsArray {
					tagArray := strings.Split(tagExpr, "=")
					if tagArray[0] == "paramname" {
						paramname = tagArray[1]
						break
					}
				}

				for _, tagExpr := range tagsArray {
					if tagExpr == "required" {
						switch structParam.paramType {
						case "string":
							fmt.Fprintln(out, "\tif params."+structParam.name+" == \"\" {")
						case "int":
							fmt.Fprintln(out, "\tif params."+structParam.name+" == 0 {")
						}
						fmt.Fprintln(out, "\t\tapiError := ApiError{Err: errors.New(\""+paramname+" must me not empty\"), HTTPStatus: http.StatusBadRequest}")
						fmt.Fprintln(out, "\t\thandleError(w, apiError)")
						fmt.Fprintln(out, "\t\treturn")
						fmt.Fprintln(out, "\t}")
						break
					}
				}

				for _, tagExpr := range tagsArray {
					tagArray := strings.Split(tagExpr, "=")
					if tagArray[0] == "default" {
						switch structParam.paramType {
						case "string":
							fmt.Fprintln(out, "\tif params." + structParam.name + " == \"\" {")
							fmt.Fprintln(out, "\t\tparams." + structParam.name + " = \"" + tagArray[1] + "\"")
						case "int":
							fmt.Fprintln(out, "\tif params." + structParam.name + " == 0 {")
							fmt.Fprintln(out, "\t\tparams." + structParam.name + " = " + tagArray[1])
						}
						fmt.Fprintln(out, "\t}")
						break
					}
				}

				for _, tagExpr := range tagsArray {
					tagArray := strings.Split(tagExpr, "=")
					switch tagArray[0] {
					case "min":
						switch structParam.paramType {
						case "string":
							fmt.Fprintln(out, "\tif len(params." + structParam.name + ") < " + tagArray[1] + " {")
							fmt.Fprintln(out, "\t\tapiErr := ApiError{Err: errors.New(\"" + paramname + " len must be >= " + tagArray[1] + "\"), HTTPStatus: http.StatusBadRequest}")
						case "int":
							fmt.Fprintln(out, "\tif params." + structParam.name + " < " + tagArray[1] + " {")
							fmt.Fprintln(out, "\t\tapiErr := ApiError{Err: errors.New(\"" + paramname + " must be >= " + tagArray[1] + "\"), HTTPStatus: http.StatusBadRequest}")
						}
						fmt.Fprintln(out, "\t\thandleError(w, apiErr)")
						fmt.Fprintln(out, "\t\treturn")
						fmt.Fprintln(out, "\t}")
					case "max":
						switch structParam.paramType {
						case "string":
							fmt.Fprintln(out, "\tif len(params." + structParam.name + ") > " + tagArray[1] + " {")
							fmt.Fprintln(out, "\t\tapiErr := ApiError{Err: errors.New(\"" + paramname + " len must be <= " + tagArray[1] + "\"), HTTPStatus: http.StatusBadRequest}")
						case "int":
							fmt.Fprintln(out, "\tif params." + structParam.name + " > " + tagArray[1] + " {")
							fmt.Fprintln(out, "\t\tapiErr := ApiError{Err: errors.New(\"" + paramname + " must be <= " + tagArray[1] + "\"), HTTPStatus: http.StatusBadRequest}")
						}
						fmt.Fprintln(out, "\t\thandleError(w, apiErr)")
						fmt.Fprintln(out, "\t\treturn")
						fmt.Fprintln(out, "\t}")
					case "enum":
						enumValues := strings.Split(tagArray[1], "|")
						enumValuesForErr := strings.Replace(tagArray[1], "|", ", ", -1)
						if len(enumValues) == 0 {
							continue
						}

						switch structParam.paramType {
						case "string":
							fmt.Fprint(out, "\tif params." + structParam.name + " != \"" + enumValues[0] + "\"")
							for i := 1; i< len(enumValues); i++ {
								fmt.Fprint(out, " &&\n\t\tparams." + structParam.name + " != \"" + enumValues[i] + "\"")
							}
						case "int":
							fmt.Fprint(out, "\tif params." + structParam.name + " != " + enumValues[0])
							for i := 1; i< len(enumValues); i++ {
								fmt.Fprint(out, " &&\n\t\tparams." + structParam.name + " != " + enumValues[i])
							}
						}
						fmt.Fprintln(out, " {")
						fmt.Fprintln(out, "\t\tapiErr := ApiError{Err: errors.New(\"" + paramname + " must be one of [" + enumValuesForErr + "]\"), HTTPStatus: http.StatusBadRequest}")
						fmt.Fprintln(out, "\t\thandleError(w, apiErr)")
						fmt.Fprintln(out, "\t\treturn")
						fmt.Fprintln(out, "\t}")
					}
				}
			}
			fmt.Fprintln(out, "\tresult, err := in." + function.name + "(context.Background(), params)")
			fmt.Fprintln(out, "\tif nil != err {")
			fmt.Fprintln(out, "\t\thandleError(w, err)")
			fmt.Fprintln(out, "\t\treturn")
			fmt.Fprintln(out, "\t}")
			fmt.Fprintln(out, "\thandleResult(w, result)")
			fmt.Fprintln(out, "}")
			fmt.Fprintln(out)
		}
	}

	fmt.Fprintln(out, "func handleError(w http.ResponseWriter, err error) {")
	//fmt.Fprint(out, "\tif apiError.HTTPStatus == http.StatusNotFound ")
	//fmt.Fprint(out, "||\n\t\t apiError.HTTPStatus == http.StatusConflict ")
	//fmt.Fprintln(out, "||\n\t\t apiError.HTTPStatus == http.StatusBadRequest {")
	fmt.Fprintln(out, "\tapiError, ok := err.(ApiError)")
	fmt.Fprintln(out, "\tif !ok {")
	fmt.Fprintln(out, "\t\tapiError = ApiError{Err: err, HTTPStatus: http.StatusInternalServerError}")
	fmt.Fprintln(out, "\t}")
	fmt.Fprintln(out, "\tvar response = make(map[string]interface{})")
	fmt.Fprintln(out, "\tresponse[\"error\"] = apiError.Err.Error()")
	fmt.Fprintln(out, "\tbody, err := json.Marshal(response)")
	fmt.Fprintln(out, "\tif nil != err {")
	fmt.Fprintln(out, "\t\tw.WriteHeader(http.StatusInternalServerError)")
	fmt.Fprintln(out, "\t\treturn")
	fmt.Fprintln(out, "\t}")
	fmt.Fprintln(out, "\tw.WriteHeader(apiError.HTTPStatus)")
	fmt.Fprintln(out, "\tw.Write(body)")
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)

	fmt.Fprintln(out, "func handleResult(w http.ResponseWriter, result interface{}) {")
	fmt.Fprintln(out, "\tvar response = make(map[string]interface{})")
	fmt.Fprintln(out, "\tresponse[\"response\"] = result")
	fmt.Fprintln(out, "\tresponse[\"error\"] = \"\"")
	fmt.Fprintln(out, "\tbody, err := json.Marshal(response)")
	fmt.Fprintln(out, "\tif nil != err {")
	fmt.Fprintln(out, "\t\tw.WriteHeader(http.StatusInternalServerError)")
	fmt.Fprintln(out, "\t\treturn")
	fmt.Fprintln(out, "\t}")
	fmt.Fprintln(out, "\tw.Write(body)")
	fmt.Fprintln(out, "}")

}