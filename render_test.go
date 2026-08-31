package artgo_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/convee/artgo"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestProtobufBindingAndRenderingUseCurrentProtobufMessages(t *testing.T) {
	message := structpb.NewStringValue("artgo")
	encoded, err := proto.Marshal(message)
	assert.Nil(t, err)

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(encoded)))
	context := &artgo.Context{Writer: httptest.NewRecorder(), Req: request}
	var decoded structpb.Value

	assert.Nil(t, context.BindProtobuf(&decoded))
	assert.Equal(t, "artgo", decoded.GetStringValue())

	recorder := httptest.NewRecorder()
	context = &artgo.Context{Writer: recorder, Req: httptest.NewRequest(http.MethodGet, "/", nil)}
	assert.Nil(t, context.RenderProtoBuf(http.StatusOK, message))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/x-protobuf", recorder.Header().Get("Content-Type"))
	assert.True(t, proto.Equal(message, &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "artgo"}}))
	assert.Equal(t, encoded, recorder.Body.Bytes())
}

func TestProtobufJSONRenderingUsesParsedMediaType(t *testing.T) {
	message := structpb.NewStringValue("artgo")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Content-Type", "APPLICATION/JSON; charset=utf-8")
	context := &artgo.Context{Writer: recorder, Req: request}

	assert.Nil(t, context.RenderProtoBuf(http.StatusOK, message))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.Equal(t, `"artgo"`, recorder.Body.String())
}

func TestProtobufRenderingRejectsTypedNilMessages(t *testing.T) {
	var message *structpb.Value
	context := &artgo.Context{Writer: httptest.NewRecorder()}

	err := context.RenderProtoBuf(http.StatusOK, message)

	assert.EqualError(t, err, "in must be a non-nil protobuf message")
}
