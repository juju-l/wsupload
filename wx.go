package main

import (
	//.v3inf "go-web/internal/domain/inf/v3" //
	//.v3service "go-web/internal/service/v3" //
	//.v3dto "go-web/internal/domain/dto/v3" //
	"context"
	"time"
	"encoding/base64"
	"strconv"
	"io"
	"encoding/json"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/grpc/reflection"
	"go-web/api/vipex.cc/wx/v1"
	"go-web/internal/domain/inf"
	"go-web/internal/service"
	"go-web/internal/domain/dto"
	"google.golang.org/grpc/credentials"
	"go-web/util/ext"
	//#"openapi/v3"
	httpbody "google.golang.org/genproto/googleapis/api/httpbody" //
	//"fmt"
	"net"
	"net/http"
	"os"
	//.v2inf "go-web/internal/domain/inf/v2" //
	//.v2service "go-web/internal/service/v2" //
	//.v2dto "go-web/internal/domain/dto/v2" //
)

var (
	//.v3svc v3inf.ApiInterface
	svc inf.ApiInterface
	//.v2svc v2inf.ApiInterface
)

type wxService struct{}

			func (rpc *wxService) Auth(ctx context.Context, req *v1.WxRequest) (*v1.WxResponse, error) {return mapRp( svc.Auth( mapRq(req) ) ), nil} /**/ //
			func (rpc *wxService) UpContents(s v1.Wx_UpContentsServer) error { var b []byte; for{ /*;*/body,err := s.Recv()
			if err==io.EOF {name:=sig();os.WriteFile(name,b,0600);dao:=mapRq(&v1.WxRequest{Sig:name});rsp:=mapRp(svc.UpContents(dao));return s.SendAndClose(rsp)}
			/*-*/if err != nil {return err};b=append(b,body.Data...) } /*;*/ } /**/ //
			func (rpc *wxService) GetAvatarUrl(ctx context.Context, req *v1.WxRequest) (*v1.WxResponse, error) {return mapRp( svc.GetAvatarUrl( mapRq(req) ) ), nil} /**/ //
			func (rpc *wxService) GetBusinessInfo(ctx context.Context, req *v1.WxRequest) (*v1.WxResponse, error) {return mapRp( svc.GetBusinessInfo( mapRq(req) ) ), nil} /**/ //
			func (rpc *wxService) UpBusinessInfo(ctx context.Context, req *v1.WxRequest) (*v1.WxResponse, error) {return mapRp( svc.UpBusinessInfo( mapRq(req) ) ), nil} /**/ //
			func (rpc *wxService) GetDetails(ctx context.Context, req *v1.WxRequest) (*v1.WxResponse, error) {return mapRp( svc.GetDetails( mapRq(req) ) ), nil} /**/ //
			func (rpc *wxService) GetOrder(ctx context.Context, req *v1.WxRequest) (*v1.WxResponse, error) {return mapRp( svc.GetOrder( mapRq(req) ) ), nil} /**/ //
			func (rpc *wxService) GetCnts(req *v1.WxRequest,s v1.Wx_GetCntsServer) error {
			return s.Send(&httpbody.HttpBody{ContentType:"",Data:( svc.GetCnts( mapRq(req) ) ).Data[req.Name].( []byte ),/*Extensions:nil*/})
			} /**/ //
			func (rpc *wxService) GetRepair(ctx context.Context, req *v1.WxRequest) (*v1.WxResponse, error) {return mapRp( svc.GetRepair( mapRq(req) ) ), nil} /**/ //

func main() {
	l, _ := net.Listen(
					"tcp", ":65443", /**/
	)
	s := grpc.NewServer(
	grpc.UnaryInterceptor(
			Authz,
	),
		grpc.Creds(ext.Must(
		credentials.
		NewServerTLSFromFile(
			"config/tls.crt",
			"config/.tls.key",
			///
			)).
		(credentials.
	TransportCredentials)),
	//grpc.MaxRecvMsgSize(
	//		1024*1024*4,
	//),
	grpc.ChainUnaryInterceptor(
			Log,
	),
	//grpc.StreamInterceptor(),
	)
	v1.RegisterWxServer(
			s, &wxService{}, /*,*/
	)
	//
		reflection.Register(s)
			//
	//if e:=s.Serve(l);e!=nil {
			//panic( e )
	//}
			//
	if e:=http.ServeTLS(
			l,
	grpcHandlerFunc(
			s, nil, reg(),
	),
			"config/tls.crt",
			"config/.tls.key",
			///
	);e!=nil {
					panic( e )/////
	}
}

func mapRq(
			req *v1.WxRequest,
	) *dto.RequestDto {
 return &dto.RequestDto {
			Code: req.Code, OpenId: req.Openid, EntId: req.Entid, Sig: req.Sig, Files: req.Files, Url: req.Url, BusinessId: req.Businessid, Name: req.Name, Num: req.Num, Type: req.Type, Description: req.Description, Detail: req.Detail, /*OrdId: req.OrdId, Repid: req.Repid,*/ Id: req.Id,
 }
}

func mapRp(
			rsp dto.ResponseDto,
		) *v1.WxResponse {
	var r = &v1.WxResponse{}
	if rsp.ErrCode == 0 { d := &structpb.Struct{};b,e := json.Marshal(rsp.Data);if e != nil {return r};d.UnmarshalJSON( b );r.Data = d } else {
			r.ErrCode=int32(rsp.ErrCode)
			/*;*/
			r.ErrMsg=rsp.ErrMsg
	}
	return r
}

//

func sig() string {
	return base64.StdEncoding.EncodeToString(
	[]byte(
	strconv.FormatInt(time.Now().UnixNano(),10),
	),
	)
}

func init() {
	//
	//.v3svc = 
	//.		&v3service. ///
	//.ApiService{}
	svc = 
			&service.
	ApiService{}
	//.v2svc = 
	//.		&v2service. ///
	//.ApiService{}
	//
}