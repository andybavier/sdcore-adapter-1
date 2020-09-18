// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0
package gnmi

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto" //nolint: staticcheck
	"github.com/openconfig/gnmi/value"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	// NOTE: This test case needs to have some models and modeldata in order
	// to run tests, so it gets these by using the sd-core synchronizers models.
	// TODO: It might be better to eventually switch to a service-independent
	// set of test models, so that this test code can remain independent of
	// any particular service.
	models "github.com/onosproject/config-models/modelplugin/aether-1.0.0/aether_1_0_0"
	"github.com/onosproject/sdcore-adapter/pkg/synchronizer/modeldata"
)

var (
	// model is the model for test config server.
	model = &Model{
		modelData:       modeldata.ModelData,
		structRootType:  reflect.TypeOf((*models.Device)(nil)),
		schemaTreeRoot:  models.SchemaTree["Device"],
		jsonUnmarshaler: models.Unmarshal,
		enumData:        map[string]map[int64]ygot.EnumDefinition{},
	}
)

func TestCapabilities(t *testing.T) {
	s, err := NewServer(model, nil, nil, nil)
	if err != nil {
		t.Fatalf("error in creating server: %v", err)
	}
	resp, err := s.Capabilities(context.TODO(), &pb.CapabilityRequest{})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if !reflect.DeepEqual(resp.GetSupportedModels(), model.modelData) {
		t.Errorf("got supported models %v\nare not the same as\nmodel supported by the server %v", resp.GetSupportedModels(), model.modelData)
	}
	if !reflect.DeepEqual(resp.GetSupportedEncodings(), supportedEncodings) {
		t.Errorf("got supported encodings %v\nare not the same as\nencodings supported by the server %v", resp.GetSupportedEncodings(), supportedEncodings)
	}
}

func TestGet(t *testing.T) {
	jsonConfigRoot := `{
		"access-profile": {
			"access-profile": [
				{
					"id": "typical-access-profile",
					"type": "internet-only",
					"filter": "allow app name",
					"description": "a typical access profile"
				}
			]
		}
	}`

	s, err := NewServer(model, []byte(jsonConfigRoot), nil, nil)
	if err != nil {
		t.Fatalf("error in creating server: %v", err)
	}

	tds := []struct {
		desc        string
		textPbPath  string
		modelData   []*pb.ModelData
		wantRetCode codes.Code
		wantRespVal interface{}
	}{{
		desc: "get access-profile",
		textPbPath: `
			elem: <name: "access-profile" >
			elem: <name: "access-profile" 
						 key: <
							 key:'id',
							 value:'typical-access-profile'
							 >
						>
			elem: <name: "filter">

		`,
		wantRetCode: codes.OK,
		wantRespVal: "allow app name",
	}}

	for _, td := range tds {
		t.Run(td.desc, func(t *testing.T) {
			runTestGet(t, s, td.textPbPath, td.wantRetCode, td.wantRespVal, td.modelData)
		})
	}
}

// runTestGet requests a path from the server by Get grpc call, and compares if
// the return code and response value are expected.
func runTestGet(t *testing.T, s *Server, textPbPath string, wantRetCode codes.Code, wantRespVal interface{}, useModels []*pb.ModelData) {
	// Send request
	var pbPath pb.Path
	if err := proto.UnmarshalText(textPbPath, &pbPath); err != nil {
		t.Fatalf("error in unmarshaling path: %v", err)
	}
	req := &pb.GetRequest{
		Path:      []*pb.Path{&pbPath},
		Encoding:  pb.Encoding_JSON_IETF,
		UseModels: useModels,
	}
	resp, err := s.Get(context.TODO(), req)

	// Check return code
	gotRetStatus, ok := status.FromError(err)
	if !ok {
		t.Fatal("got a non-grpc error from grpc call")
	}
	if gotRetStatus.Code() != wantRetCode {
		t.Fatalf("got return code %v, want %v", gotRetStatus.Code(), wantRetCode)
	}

	// Check response value
	var gotVal interface{}
	if resp != nil {
		notifs := resp.GetNotification()
		if len(notifs) != 1 {
			t.Fatalf("got %d notifications, want 1", len(notifs))
		}
		updates := notifs[0].GetUpdate()
		if len(updates) != 1 {
			t.Fatalf("got %d updates in the notification, want 1", len(updates))
		}
		val := updates[0].GetVal()
		if val.GetJsonIetfVal() == nil {
			gotVal, err = value.ToScalar(val)
			if err != nil {
				t.Errorf("got: %v, want a scalar value", gotVal)
			}
		} else {
			// Unmarshal json data to gotVal container for comparison
			if err := json.Unmarshal(val.GetJsonIetfVal(), &gotVal); err != nil {
				t.Fatalf("error in unmarshaling IETF JSON data to json container: %v", err)
			}
			var wantJSONStruct interface{}
			if err := json.Unmarshal([]byte(wantRespVal.(string)), &wantJSONStruct); err != nil {
				t.Fatalf("error in unmarshaling IETF JSON data to json container: %v", err)
			}
			wantRespVal = wantJSONStruct
		}
	}

	if !reflect.DeepEqual(gotVal, wantRespVal) {
		t.Errorf("got: %v (%T),\nwant %v (%T)", gotVal, gotVal, wantRespVal, wantRespVal)
	}
}