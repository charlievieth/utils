package logging

import (
	"context"
	"math/rand/v2"
	"net/http"
	"path"

	"go.uber.org/zap"
)

type ctxKey int8

const ctxLogKey ctxKey = 0

type Middleware struct {
	log  *zap.Logger
	next http.Handler
}

// var rr = rand.New(rand.NewPCG(randUint64(), randUint64()))

func NewMiddleware(log *zap.Logger, next http.Handler) http.Handler {
	return &Middleware{log: log, next: next}
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := m.log.With(
		zap.Uint64("id", rand.Uint64()),
		zap.String("url", path.Join(r.URL.Host, r.URL.Path)),
		zap.String("method", r.Method),
	)
	m.next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxLogKey, log)))
}

func GetLogger(r *http.Request) (log *zap.Logger) {
	if v := r.Context().Value(ctxLogKey); v != nil {
		log = v.(*zap.Logger)
	} else {
		log = zap.L()
	}

	return log
}

// func randUint64() uint64 {
// 	bi, err := crand.Int(crand.Reader, new(big.Int).SetUint64(math.MaxUint64))
// 	if err != nil {
// 		panic(err)
// 	}
// 	return bi.Uint64()
// }
