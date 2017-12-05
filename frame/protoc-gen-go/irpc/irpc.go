// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2015 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Package irpc outputs iRPC service descriptions in Go code.
// It runs as a plugin for the Go protocol buffer compiler plugin.
// It is linked in to protoc-gen-go.
package irpc

import (
	"fmt"
	"strconv"
	"strings"

	pb "github.com/kwins/iceberg/frame/protoc-gen-go/descriptor"
	"github.com/kwins/iceberg/frame/protoc-gen-go/generator"
)

// generatedCodeVersion indicates a version of the generated code.
// It is incremented whenever an incompatibility between the generated code and
// the grpc package is introduced; the generated code references
// a constant, grpc.SupportPackageIsVersionN (where N is generatedCodeVersion).
const generatedCodeVersion = 4

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath = "context"
)

func init() {
	generator.RegisterPlugin(new(irpc))
}

// grpc is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for gRPC support.
type irpc struct {
	gen *generator.Generator
}

// Name returns the name of this plugin, "irpc".
func (ig *irpc) Name() string {
	return "irpc"
}

// The names for packages imported in the generated code.
// They may vary from the final path component of the import path
// if the name is used by other packages.
var (
	contextPkg string
)

// Init initializes the plugin.
func (ig *irpc) Init(gen *generator.Generator) {
	ig.gen = gen
	contextPkg = generator.RegisterUniquePackageName("context", nil)
	// irpcPkg = generator.RegisterUniquePackageName("irpc", nil)
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (ig *irpc) objectNamed(name string) generator.Object {
	ig.gen.RecordTypeUse(name)
	return ig.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (ig *irpc) typeName(str string) string {
	return ig.gen.TypeName(ig.objectNamed(str))
}

// P forwards to g.gen.P.
func (ig *irpc) P(args ...interface{}) { ig.gen.P(args...) }

// Generate generates code for the services in the given file.
func (ig *irpc) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	ig.P("// Reference imports to suppress errors if they are not otherwise used.")
	ig.P("var _ ", contextPkg, ".Context")
	ig.P()

	// Assert version compatibility.
	ig.P("// This is a compile-time assertion to ensure that this generated file")
	ig.P("// is compatible with the grpc package it is being compiled against.")
	// ig.P("const _ = irpc.SupportPackageIsVersion", generatedCodeVersion)
	ig.P()

	for i, service := range file.FileDescriptorProto.Service {
		ig.generateService(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (ig *irpc) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	ig.P("import (")
	ig.P(strconv.Quote("context"))
	ig.P(strconv.Quote("github.com/kwins/iceberg/frame"))
	ig.P(strconv.Quote("github.com/kwins/iceberg/frame/config"))
	ig.P(strconv.Quote("github.com/kwins/iceberg/frame/protocol"))
	ig.P(")")
	ig.P()
}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
// TODO: do we need any in gRPC?
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// generateService generates all the code for the named service.
func (ig *irpc) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	path := fmt.Sprintf("6,%d", index) // 6 means service.

	origServName := service.GetName()
	fullServName := origServName
	if pkg := file.GetPackage(); pkg != "" {
		fullServName = pkg + "." + fullServName
	}
	servName := generator.CamelCase(origServName)

	ig.P()
	ig.P("// Client API for ", servName, " service")
	ig.P("// iceberg server version,relation to server uri.")
	ig.P("var ", unexport(origServName)+"_version =  frame.SrvVersionName[frame.SV1]")
	ig.P()

	// Client interface.
	for i, method := range service.Method {
		ig.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.

		ig.P("func ", ig.generateClientSignature(servName, method), " {")
		if pkg := file.GetOptions().GetGoPackage(); pkg == "" {
			ig.P("task, err := frame.ReadyTask(ctx, ", strconv.Quote(method.GetName()), ", ", strconv.Quote(unexport(servName)), " ,in)")
		} else {
			ig.P("task, err := frame.ReadyTask(ctx, ", strconv.Quote(method.GetName()), ", ", strconv.Quote(pkg+"/"+unexport(servName)), " ,in)")
		}

		ig.P("if err != nil {")
		ig.P("	return nil, err")
		ig.P("}")

		ig.P("if span := frame.SpanWithTask(ctx, task); span != nil {")
		ig.P("	defer span.Finish()")
		ig.P("}")

		ig.P("back, err := frame.DeliverTo(task)")
		ig.P("if err != nil {")
		ig.P("	return nil, err")
		ig.P("}")

		ig.P()
		ig.P("var out ", ig.typeName(method.GetOutputType()))
		ig.P("if err := protocol.Unpack(task.GetFormat(), back.GetBody(), &out); err != nil {")
		ig.P("	return nil, err")
		ig.P("}")
		ig.P("return &out, nil")
		ig.P("}")
	}

	// Server interface
	ig.P("// ", servName, "Server Server API for Hello service")
	ig.P("type ", servName, "Server interface{")
	for _, method := range service.Method {
		ig.P()
		ig.P(ig.generateClientSignature(servName, method))
		ig.P()
	}
	ig.P("}")

	ig.P("// Register", servName, "Server register ", servName, "Server with etcd info")
	ig.P("func Register", servName, "Server(srv ", servName, "Server, cfg *config.BaseCfg) {")
	ig.P("frame.RegisterAndServe(&", unexport(servName), "ServerDesc, srv, cfg)")
	ig.P("}")

	for _, method := range service.Method {
		ig.P("// ", unexport(servName), " server ", method.GetName(), " handler")
		ig.P("func ", unexport(servName), method.GetName(), "Handler(srv interface{}, ctx context.Context, format protocol.RestfulFormat, data []byte) ([]byte, error) {")
		ig.P("var in ", ig.typeName(method.GetInputType()))
		ig.P("if err := protocol.Unpack(format, data, &in); err != nil {")
		ig.P("	return nil, err")
		ig.P("}")
		ig.P()
		ig.P(unexport(servName), "Resp,err := srv.(", servName, "Server).", method.GetName(), "(ctx, &in)")
		ig.P("if err != nil {")
		ig.P("	return nil, err")
		ig.P("}")
		ig.P("b, err := protocol.Pack(format, ", unexport(servName), "Resp)")
		ig.P("if err != nil {")
		ig.P("	return nil, err")
		ig.P("}")
		ig.P("return b, nil")
		ig.P("}")
	}

	ig.P("// ", unexport(servName), " server describe")

	ig.P("var ", unexport(servName), "ServerDesc = frame.ServiceDesc {")
	ig.P("Version:", unexport(origServName), "_version,")
	ig.P("ServiceName:", strconv.Quote(servName), ",")
	ig.P("HandlerType:", "(*", servName, "Server)(nil),")
	ig.P("Methods: []frame.MethodDesc{")

	for _, method := range service.Method {
		ig.P("{")
		ig.P("MethodName: ", strconv.Quote(method.GetName()), ",")
		ig.P("Handler: ", unexport(servName)+method.GetName()+"Handler,")
		ig.P("},")
	}

	ig.P("},")
	ig.P("ServiceURI: []string{")
	if pkg := file.GetOptions().GetGoPackage(); pkg == "" {
		ig.P(strconv.Quote("/services/"), " + ", unexport(origServName), "_version + ", strconv.Quote("/"+unexport(servName)), ",")
	} else {
		ig.P(strconv.Quote("/services/"), " + ", unexport(origServName), "_version + ", strconv.Quote("/"+pkg+"/"+unexport(servName)), ",")
	}
	ig.P("},")

	ig.P("Metadata: ", strconv.Quote(fullServName), ",")
	ig.P("}")
}

// generateClientSignature returns the client-side signature for a method.
func (ig *irpc) generateClientSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}
	reqArg := ", in *" + ig.typeName(method.GetInputType())
	if method.GetClientStreaming() {
		reqArg = ""
	}
	respName := "*" + ig.typeName(method.GetOutputType())
	if method.GetServerStreaming() || method.GetClientStreaming() {
		respName = servName + "_" + generator.CamelCase(origMethName) + "Client"
	}
	return fmt.Sprintf("%s(ctx %s.Context%s) (%s, error)", methName, contextPkg, reqArg, respName)
}
