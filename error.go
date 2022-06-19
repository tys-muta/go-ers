package error

// 下記を考慮したエラーパッケージ.
//
// - gRPC の仕様
// ラップした場合でも, ラップされたエラーをたどって codes.Code を取得可能
// - スタックトレース
// %w でラップしただけではスタックトレースが行えないため, xerrors パッケージを使ってエラー発生時の原因補足を行い易く
// - 表示用メッセージ
// エラーメッセージはそのまま表示せず, 表示用として設定可能

import (
	"errors"
	"fmt"

	"golang.org/x/xerrors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// 制御用
	ErrWrap = New(codes.OK, "", "")

	// gRPC のエラーに基づいたエラー
	ErrCanceled           = /* HTTP: 499 gRPC:  1 */ New(codes.Canceled, "Canceled", "処理がキャンセルされました。")
	ErrUnknown            = /* HTTP: 500 gRPC:  2 */ New(codes.Unknown, "Unknown", "不明なエラーが発生しました。")
	ErrInvalidArgument    = /* HTTP: 400 gRPC:  3 */ New(codes.InvalidArgument, "InvalidArgument", "入力値が不正です。")
	ErrDeadlineExceeded   = /* HTTP: 504 gRPC:  4 */ New(codes.DeadlineExceeded, "DeadlineExceeded", "処理がタイムアウトしました。")
	ErrNotFound           = /* HTTP: 404 gRPC:  5 */ New(codes.NotFound, "NotFound", "存在しないデータへの参照が発生しています。")
	ErrAlreadyExists      = /* HTTP: 409 gRPC:  6 */ New(codes.AlreadyExists, "AlreadyExists", "データが既に存在します。")
	ErrPermissionDenied   = /* HTTP: 403 gRPC:  7 */ New(codes.PermissionDenied, "PermissionDenied", "必要な権限がありません。")
	ErrResourceExhausted  = /* HTTP: 429 gRPC:  8 */ New(codes.ResourceExhausted, "ResourceExhausted", "処理限界を超えています。")
	ErrFailedPrecondition = /* HTTP: 400 gRPC:  9 */ New(codes.FailedPrecondition, "FailedPrecondition", "必要な条件を満たしていません。")
	ErrAborted            = /* HTTP: 409 gRPC: 10 */ New(codes.Aborted, "Aborted", "操作が中断されました。")
	ErrOutOfRange         = /* HTTP: 400 gRPC: 11 */ New(codes.OutOfRange, "OutOfRange", "入力値が有効範囲外です。")
	ErrUnimplemented      = /* HTTP: 501 gRPC: 12 */ New(codes.Unimplemented, "Unimplemented", "サポートされていません。")
	ErrInternal           = /* HTTP: 500 gRPC: 13 */ New(codes.Internal, "Internal", "システム内部でエラーが発生しました。")
	ErrUnavailable        = /* HTTP: 503 gRPC: 14 */ New(codes.Unavailable, "Unavailable", "システムは現在利用できません。")
	ErrDataLoss           = /* HTTP: 500 gRPC: 15 */ New(codes.DataLoss, "DataLoss", "修復不能なデータの欠損が生じました。")
	ErrUnauthenticated    = /* HTTP: 401 gRPC: 16 */ New(codes.Unauthenticated, "Unauthenticated", "認証できませんでした。")
)

var (
	// W 関数は, NewWrap 関数のエイリアス.
	W = NewWrap
)

type Error struct {
	error
	code    codes.Code
	reason  string
	message string
	trace   Trace
	frame   xerrors.Frame
	domain  string
}

func New(code codes.Code, reason string, message string) *Error {
	return &Error{
		code:    code,
		reason:  reason,
		message: message,
	}
}

func (e *Error) New(v interface{}) error {
	err := &Error{
		code:    e.code,
		reason:  e.reason,
		message: e.message,
		frame:   xerrors.Caller(1),
		trace:   newTrace(v),
	}
	return err
}

func NewWrap(err error, options ...Option) error {
	v := &Error{
		error:   err,
		code:    ErrWrap.code,
		reason:  ErrWrap.reason,
		message: ErrWrap.message,
		frame:   xerrors.Caller(1),
	}
	o := NewOptions(options...)
	if o.Trace != nil {
		v.trace = *o.Trace
	}
	return v
}

func Is(err error, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

func (e *Error) Unwrap() error {
	return e.error
}

func (e *Error) Is(target error) bool {
	if err, ok := target.(*Error); ok && e.code == err.code && e.reason == err.reason {
		return true
	}
	return false
}

func (e *Error) As(target interface{}) bool {
	if err, ok := target.(**Error); ok {
		(*err).error = e.error
		(*err).code = e.code
		(*err).reason = e.reason
		(*err).message = e.message
		(*err).trace = e.trace
		(*err).frame = e.frame
		return true
	}
	return false
}

func (e *Error) Format(state fmt.State, rune rune) {
	xerrors.FormatError(e, state, rune)
}

func (e *Error) FormatError(p xerrors.Printer) (next error) {
	p.Print(e.trace.Dump())
	e.frame.Format(p)
	return e.error
}

func (e *Error) Error() string {
	if e.error != nil {
		return e.error.Error()
	}
	return fmt.Sprintf("%v", e)
}

func (e *Error) WithDomain(domain string) *Error {
	e.domain = domain
	return e
}

func (e *Error) GRPCStatus() *status.Status {
	grpcStatus := status.New(e.Code(), e.Message())
	grpcStatus, _ = grpcStatus.WithDetails(&errdetails.ErrorInfo{
		Reason: e.Reason(),
		Domain: e.Domain(),
		Metadata: map[string]string{
			"Trace": fmt.Sprintf("%v", e),
		},
	})
	return grpcStatus
}

func (e *Error) Code() codes.Code {
	if e.error != nil {
		if err, ok := e.error.(interface{ GRPCStatus() *status.Status }); ok {
			return err.GRPCStatus().Code()
		}
		if err, ok := e.error.(interface{ Code() codes.Code }); ok {
			return err.Code()
		}
	}
	return e.code
}

func (e *Error) Message() string {
	if e.message != "" {
		return e.message
	}
	if e.error != nil {
		if err, ok := e.error.(interface{ Message() string }); ok {
			return err.Message()
		}
		if err, ok := e.error.(interface{ GRPCStatus() *status.Status }); ok {
			switch err.GRPCStatus().Code() {
			case codes.Canceled:
				return ErrCanceled.message
			case codes.Unknown:
				return ErrUnknown.message
			case codes.InvalidArgument:
				return ErrInvalidArgument.message
			case codes.DeadlineExceeded:
				return ErrDeadlineExceeded.message
			case codes.NotFound:
				return ErrNotFound.message
			case codes.AlreadyExists:
				return ErrAlreadyExists.message
			case codes.PermissionDenied:
				return ErrPermissionDenied.reason
			case codes.ResourceExhausted:
				return ErrResourceExhausted.message
			case codes.FailedPrecondition:
				return ErrFailedPrecondition.message
			case codes.Aborted:
				return ErrAborted.message
			case codes.OutOfRange:
				return ErrOutOfRange.message
			case codes.Unimplemented:
				return ErrUnimplemented.message
			case codes.Internal:
				return ErrInternal.message
			case codes.Unavailable:
				return ErrUnavailable.message
			case codes.DataLoss:
				return ErrDataLoss.message
			case codes.Unauthenticated:
				return ErrUnauthenticated.message
			}
		}
	}
	return ""
}

func (e *Error) Reason() string {
	if e.reason != "" {
		return e.reason
	}
	if e.error != nil {
		if err, ok := e.error.(interface{ Reason() string }); ok {
			return err.Reason()
		}
	}
	return ""
}

func (e *Error) Domain() string {
	if e.domain != "" {
		return e.domain
	}
	if e.error != nil {
		if err, ok := e.error.(interface{ Domain() string }); ok {
			return err.Domain()
		}
	}
	return ""
}