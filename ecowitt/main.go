package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"golang.org/x/term"

	"github.com/charlievieth/utils/ecowitt/pkg/middleware/logging"
)

// WARN: make sure we like this namespace
const Namespace = "nyc_weather"

var _ = promhttp.Handler

func init() {
	// Logging
	var config zap.Config
	if term.IsTerminal(int(os.Stdout.Fd())) {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}
	config.OutputPaths = []string{"stdout"}
	log, err := config.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(log.WithOptions(zap.AddStacktrace(zap.ErrorLevel)))

	// Prometheus
	prometheus.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{
		Namespace: Namespace,
	}))

	// TODO: we probably don't need this, but if we do add it
	// then we should include some CPU stats.
	//
	// prometheus.MustRegister(
	// 	collectors.NewGoCollector(
	// 		collectors.WithGoCollectorRuntimeMetrics(
	// 			collectors.GoRuntimeMetricsRule{
	// 				Matcher: regexp.MustCompile(`^/memory/classes/heap/(free|objects|released|stacks|unused):bytes`),
	// 			},
	// 			collectors.GoRuntimeMetricsRule{
	// 				Matcher: regexp.MustCompile(`^/sched/goroutines:goroutines`),
	// 			},
	// 		),
	// 	),
	// )
}

type Config struct {
	PassKeys map[string]bool // Use this for validation
	conn     *pgx.Conn
	log      *zap.Logger
}

type GaugeSetter struct {
	gauge prometheus.Gauge
	fn    func(d *FormData) float64
}

func newGaugeSetter(name string, fn func(d *FormData) float64) *GaugeSetter {
	// Need to delay initializing the zap.Logger until the init
	// function runs, which occurs after this is called since
	// this is used in variable assignment.
	var log *zap.Logger
	initLogger := sync.OnceFunc(func() {
		log = zap.L().Named("weather_gauge")
	})
	return &GaugeSetter{
		gauge: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      name,
		}),
		fn: func(d *FormData) float64 {
			initLogger()
			v := fn(d)
			if log.Level().Enabled(zap.DebugLevel) {
				log.Debug(name, zap.Float64("value", v))
			}
			return v
		},
	}
}

func (s *GaugeSetter) Update(d *FormData) {
	if s.fn != nil && d != nil {
		s.gauge.Set(s.fn(d))
	}
}

var metrics = []*GaugeSetter{
	newGaugeSetter("outdoor_humidity", (*FormData).GetOutdoorHumidity),
	newGaugeSetter("outdoor_temperature", (*FormData).GetOutdoorTemperature),
	newGaugeSetter("indoor_humidity", (*FormData).GetIndoorHumidity),
	newGaugeSetter("indoor_temperature", (*FormData).GetIndoorTemperature),
	newGaugeSetter("solar_irradiance", (*FormData).GetSolarIrradiance),
	newGaugeSetter("solar_uvi", (*FormData).GetUVI),
	newGaugeSetter("rainfall_daily", (*FormData).GetRainfallDaily),
	newGaugeSetter("rainfall_event", (*FormData).GetRainfallEvent),
	newGaugeSetter("rainfall_hourly", (*FormData).GetRainfallHourly),
	newGaugeSetter("rainfall_monthly", (*FormData).GetRainfallMonthly),
	newGaugeSetter("rainfall_rate", (*FormData).GetRainfallRate),
	newGaugeSetter("rainfall_state", (*FormData).GetRainfallState),
	newGaugeSetter("rainfall_weekly", (*FormData).GetRainfallWeekly),
	newGaugeSetter("rainfall_yearly", (*FormData).GetRainfallYearly),
	newGaugeSetter("wind_direction", (*FormData).GetWindDirection),
	newGaugeSetter("wind_gust", (*FormData).GetWindGust),
	newGaugeSetter("wind_speed", (*FormData).GetWindSpeed),
	newGaugeSetter("wind_max_daily_gust", (*FormData).GetWindMaxDailyGust),
	newGaugeSetter("pressure_absolute", (*FormData).GetPressureAbsolute),
	newGaugeSetter("pressure_relative", (*FormData).GetPressureRelative),
	newGaugeSetter("battery_voltage", (*FormData).GetBatteryVoltage),
	newGaugeSetter("capacitor_voltage", (*FormData).GetCapacitorVoltage),
}

func UpdateMetrics(d *FormData) error {
	for _, m := range metrics {
		m.Update(d)
	}
	return nil
}

// // WARN: need to also persist this to sqlite3
// const createTableStmt = `
// CREATE TABLE IF NOT EXISTS nyc_weather(
// 	created_at          INTEGER PRIMARY KEY,
// 	outdoor_humidity    REAL NOT NULL,
// 	outdoor_temperature REAL NOT NULL,
// 	indoor_humidity     REAL NOT NULL,
// 	indoor_temperature  REAL NOT NULL,
// 	solar_irradiance    REAL NOT NULL,
// 	solar_uvi           INTEGER NOT NULL,
// 	rainfall_daily      REAL NOT NULL,
// 	rainfall_event      REAL NOT NULL,
// 	rainfall_hourly     REAL NOT NULL,
// 	rainfall_monthly    REAL NOT NULL,
// 	rainfall_rate       REAL NOT NULL,
// 	rainfall_state      INTEGER NOT NULL,
// 	rainfall_weekly     REAL NOT NULL,
// 	rainfall_yearly     REAL NOT NULL,
// 	wind_direction      REAL NOT NULL,
// 	wind_gust           REAL NOT NULL,
// 	wind_speed          REAL NOT NULL,
// 	wind_max_daily_gust REAL NOT NULL,
// 	pressure_absolute   REAL NOT NULL,
// 	pressure_relative   REAL NOT NULL,
// 	battery_voltage     REAL NOT NULL,
// 	capacitor_voltage   REAL NOT NULLL
// ) WITHOUT ROWID;`

const createTableStmt = `
CREATE TABLE IF NOT EXISTS weather_passkeys(
	id  SERIAL PRIMARY KEY,
	key CHARACTER(32) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS weather(
	id                  BIGSERIAL PRIMARY KEY,
	created_at          TIMESTAMP WITH TIME ZONE NOT NULL,
	passkey             INTEGER NOT NULL references weather_passkeys(id),
	outdoor_humidity    REAL NOT NULL,
	outdoor_temperature REAL NOT NULL,
	indoor_humidity     REAL NOT NULL,
	indoor_temperature  REAL NOT NULL,
	solar_irradiance    REAL NOT NULL,
	solar_uvi           INTEGER NOT NULL,
	rainfall_daily      REAL NOT NULL,
	rainfall_event      REAL NOT NULL,
	rainfall_hourly     REAL NOT NULL,
	rainfall_monthly    REAL NOT NULL,
	rainfall_rate       REAL NOT NULL,
	rainfall_state      INTEGER NOT NULL,
	rainfall_weekly     REAL NOT NULL,
	rainfall_yearly     REAL NOT NULL,
	wind_direction      REAL NOT NULL,
	wind_gust           REAL NOT NULL,
	wind_speed          REAL NOT NULL,
	wind_max_daily_gust REAL NOT NULL,
	pressure_absolute   REAL NOT NULL,
	pressure_relative   REAL NOT NULL,
	battery_voltage     REAL NOT NULL,
	capacitor_voltage   REAL NOT NULL
);
`

func InsertRow(ctx context.Context, db *pgx.Conn, d *FormData) error {
	const (
		selectPassKeyStmt = `SELECT id FROM weather_passkeys WHERE key = $1;`
		insertPassKeyStmt = `INSERT INTO weather_passkeys(key) VALUES ($1) RETURNING id;`
	)
	var passkey int64
	if err := db.QueryRow(ctx, selectPassKeyStmt, d.PassKey).Scan(&passkey); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			panic("HERE 1")
			return err
		}
		// Add row
		if err := db.QueryRow(ctx, insertPassKeyStmt, d.PassKey).Scan(&passkey); err != nil {
			panic("HERE 2")
			return err
		}
	}
	const stmt = `
	INSERT INTO weather(
		created_at,
		passkey,
		outdoor_humidity,
		outdoor_temperature,
		indoor_humidity,
		indoor_temperature,
		solar_irradiance,
		solar_uvi,
		rainfall_daily,
		rainfall_event,
		rainfall_hourly,
		rainfall_monthly,
		rainfall_rate,
		rainfall_state,
		rainfall_weekly,
		rainfall_yearly,
		wind_direction,
		wind_gust,
		wind_speed,
		wind_max_daily_gust,
		pressure_absolute,
		pressure_relative,
		battery_voltage,
		capacitor_voltage
	) VALUES (
		$1, $2, $3, $4,
		$5, $6, $7, $8,
		$9, $10, $11, $12,
		$13, $14, $15, $16,
		$17, $18, $19, $20,
		$21, $22, $23, $24
	);`
	_, err := db.Exec(ctx, stmt,
		time.Now(),
		passkey,
		d.OutdoorHumidity,
		d.OutdoorTemperature,
		d.IndoorHumidity,
		d.IndoorTemperature,
		d.SolarIrradiance,
		int64(d.UVI),
		d.RainfallDaily,
		d.RainfallEvent,
		d.RainfallHourly,
		d.RainfallMonthly,
		d.RainfallRate,
		int64(d.RainfallState),
		d.RainfallWeekly,
		d.RainfallYearly,
		d.WindDirection,
		d.WindGust,
		d.WindSpeed,
		d.WindMaxDailyGust,
		d.PressureAbsolute,
		d.PressureRelative,
		d.BatteryVoltage,
		d.CapacitorVoltage,
	)
	return err
}

// func InsertRow(ctx context.Context, db *sql.DB, d *FormData) error {
// 	const stmt = `
// 	INSERT INTO nyc_weather(
// 		created_at,
// 		outdoor_humidity,
// 		outdoor_temperature,
// 		indoor_humidity,
// 		indoor_temperature,
// 		solar_irradiance,
// 		solar_uvi,
// 		rainfall_daily,
// 		rainfall_event,
// 		rainfall_hourly,
// 		rainfall_monthly,
// 		rainfall_rate,
// 		rainfall_state,
// 		rainfall_weekly,
// 		rainfall_yearly,
// 		wind_direction,
// 		wind_gust,
// 		wind_speed,
// 		wind_max_daily_gust,
// 		pressure_absolute,
// 		pressure_relative,
// 		battery_voltage,
// 		capacitor_voltage
// 	) VALUES (? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ?);`
// 	db.ExecContext(ctx, stmt,
// 		time.Now().Unix(),
// 		d.OutdoorHumidity,
// 		d.OutdoorTemperature,
// 		d.IndoorHumidity,
// 		d.IndoorTemperature,
// 		d.SolarIrradiance,
// 		int64(d.UVI),
// 		d.RainfallDaily,
// 		d.RainfallEvent,
// 		d.RainfallHourly,
// 		d.RainfallMonthly,
// 		d.RainfallRate,
// 		int64(d.RainfallState),
// 		d.RainfallWeekly,
// 		d.RainfallYearly,
// 		d.WindDirection,
// 		d.WindGust,
// 		d.WindSpeed,
// 		d.WindMaxDailyGust,
// 		d.PressureAbsolute,
// 		d.PressureRelative,
// 		d.BatteryVoltage,
// 		d.CapacitorVoltage,
// 	)
// 	return nil
// }

type FormData struct {
	PassKey            string  `json:"pass_key"`
	OutdoorHumidity    float64 `json:"outdoor_humidity" prom:"outdoor_humidity"`
	OutdoorTemperature float64 `json:"outdoor_temperature" prom:"outdoor_temperature"`
	IndoorHumidity     float64 `json:"indoor_humidity" prom:"indoor_humidity"`
	IndoorTemperature  float64 `json:"indoor_temperature" prom:"indoor_temperature"`
	SolarIrradiance    float64 `json:"solar_irradiance" prom:"solar_irradiance"`
	UVI                float64 `json:"solar_uvi" prom:"solar_uvi"` // WARN: this is really an int
	RainfallDaily      float64 `json:"rainfall_daily" prom:"rainfall_daily"`
	RainfallEvent      float64 `json:"rainfall_event" prom:"rainfall_event"`
	RainfallHourly     float64 `json:"rainfall_hourly" prom:"rainfall_hourly"`
	RainfallMonthly    float64 `json:"rainfall_monthly" prom:"rainfall_monthly"`
	RainfallRate       float64 `json:"rainfall_rate" prom:"rainfall_rate"`
	RainfallState      float64 `json:"rainfall_state" prom:"rainfall_state"` // WARN: this is really an int
	RainfallWeekly     float64 `json:"rainfall_weekly" prom:"rainfall_weekly"`
	RainfallYearly     float64 `json:"rainfall_yearly" prom:"rainfall_yearly"`
	WindDirection      float64 `json:"wind_direction" prom:"wind_direction"`
	WindGust           float64 `json:"wind_gust" prom:"wind_gust"`
	WindSpeed          float64 `json:"wind_speed" prom:"wind_speed"`
	WindMaxDailyGust   float64 `json:"wind_max_daily_gust" prom:"wind_max_daily_gust"`
	PressureAbsolute   float64 `json:"pressure_absolute" prom:"pressure_absolute"`
	PressureRelative   float64 `json:"pressure_relative" prom:"pressure_relative"`
	BatteryVoltage     float64 `json:"battery_voltage" prom:"battery_voltage"`
	CapacitorVoltage   float64 `json:"capacitor_voltage" prom:"capacitor_voltage"`
}

func (d *FormData) GetOutdoorHumidity() float64    { return d.OutdoorHumidity }
func (d *FormData) GetOutdoorTemperature() float64 { return d.OutdoorTemperature }
func (d *FormData) GetIndoorHumidity() float64     { return d.IndoorHumidity }
func (d *FormData) GetIndoorTemperature() float64  { return d.IndoorTemperature }
func (d *FormData) GetSolarIrradiance() float64    { return d.SolarIrradiance }
func (d *FormData) GetUVI() float64                { return d.UVI }
func (d *FormData) GetRainfallDaily() float64      { return d.RainfallDaily }
func (d *FormData) GetRainfallEvent() float64      { return d.RainfallEvent }
func (d *FormData) GetRainfallHourly() float64     { return d.RainfallHourly }
func (d *FormData) GetRainfallMonthly() float64    { return d.RainfallMonthly }
func (d *FormData) GetRainfallRate() float64       { return d.RainfallRate }
func (d *FormData) GetRainfallState() float64      { return d.RainfallState }
func (d *FormData) GetRainfallWeekly() float64     { return d.RainfallWeekly }
func (d *FormData) GetRainfallYearly() float64     { return d.RainfallYearly }
func (d *FormData) GetWindDirection() float64      { return d.WindDirection }
func (d *FormData) GetWindGust() float64           { return d.WindGust }
func (d *FormData) GetWindSpeed() float64          { return d.WindSpeed }
func (d *FormData) GetWindMaxDailyGust() float64   { return d.WindMaxDailyGust }
func (d *FormData) GetPressureAbsolute() float64   { return d.PressureAbsolute }
func (d *FormData) GetPressureRelative() float64   { return d.PressureRelative }
func (d *FormData) GetBatteryVoltage() float64     { return d.BatteryVoltage }
func (d *FormData) GetCapacitorVoltage() float64   { return d.CapacitorVoltage }

func ParseFormData(form url.Values) (*FormData, error) {
	d := FormData{
		PassKey: form.Get("PASSKEY"),
	}

	var targets = []struct {
		dst *float64
		key string
	}{
		{&d.OutdoorHumidity, "humidity"},
		{&d.OutdoorTemperature, "tempf"},
		{&d.IndoorHumidity, "humidityin"},
		{&d.IndoorTemperature, "tempinf"},
		{&d.SolarIrradiance, "solarradiation"},
		{&d.UVI, "uv"},
		{&d.RainfallDaily, "drain_piezo"},
		{&d.RainfallEvent, "erain_piezo"},
		{&d.RainfallHourly, "hrain_piezo"},
		{&d.RainfallMonthly, "mrain_piezo"},
		{&d.RainfallRate, "rrain_piezo"},
		{&d.RainfallState, "srain_piezo"},
		{&d.RainfallWeekly, "wrain_piezo"},
		{&d.RainfallYearly, "yrain_piezo"},
		{&d.WindDirection, "winddir"},
		{&d.WindGust, "windgustmph"},
		{&d.WindSpeed, "windspeedmph"},
		{&d.WindMaxDailyGust, "maxdailygust"},
		{&d.PressureAbsolute, "baromabsin"},
		{&d.PressureRelative, "baromrelin"},
		{&d.BatteryVoltage, "wh90batt"},
		{&d.CapacitorVoltage, "ws90cap_volt"},
	}
	var errs []error
	for _, t := range targets {
		if !form.Has(t.key) {
			errs = append(errs, errors.New("missing form value: "+t.key))
			continue
		}
		v, err := strconv.ParseFloat(form.Get(t.key), 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("parsing form value: %s: %w", t.key, err))
			continue
		}
		*t.dst = v
	}
	return &d, errors.Join(errs...)
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		r.Body.Close()
	}
	http.Error(w, "not found: "+path.Join(r.URL.Host, r.URL.Path), 404)
}

// TODO: use [http.MaxBytesReader]
func Routes() *http.ServeMux {
	log := zap.L().Named("http")
	mux := http.NewServeMux()
	handle := func(pattern string, handler http.Handler) {
		log = log.With(zap.String("pattern", pattern))
		mux.Handle(pattern, logging.NewMiddleware(log, handler))
	}
	handle("POST /data/report/{$}", Handler(HandleEcowittPost))
	// mux.Handle("POST /data/report/{$}", logging.NewMiddleware(log, Handler(HandleEcowittPost)))
	handle("/", http.HandlerFunc(NotFoundHandler))
	// mux.Handle("/", logging.NewMiddleware(log, http.HandlerFunc(NotFoundHandler)))
	// mux.Handle("/", logging.NewMiddleware(log, http.HandlerFunc(NotFoundHandler)))
	opts := promhttp.HandlerOpts{
		ErrorLog:         zap.NewStdLog(log.Named("prom")),
		ErrorHandling:    promhttp.HTTPErrorOnError,
		Timeout:          30 * time.Second,
		ProcessStartTime: time.Now(),
	}
	handle("/metrics", promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer, promhttp.HandlerFor(prometheus.DefaultGatherer, opts),
	))
	// mux.Handle("/metrics", promhttp.InstrumentMetricHandler(
	// 	prometheus.DefaultRegisterer, promhttp.HandlerFor(prometheus.DefaultGatherer, opts),
	// ))
	return mux
}

func NewServer(ctx context.Context, addr string, handler http.Handler) *http.Server {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      15 * time.Second,
		MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
		ErrorLog: zap.NewStdLog(zap.L().Named("http.server").WithOptions(
			zap.AddStacktrace(zap.FatalLevel),
		)),
		BaseContext: func(l net.Listener) context.Context {
			fmt.Println("BaseContext:", l)
			return ctx
		},
	}
	return srv
}

func closeBody(body io.ReadCloser) {
	if body != nil {
		defer body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(body, 64*1024))
	}
}

type HTTPError struct {
	Code int
	Err  error
}

func NewHTTPError(code int, err error) *HTTPError {
	if code == 0 {
		code = http.StatusInternalServerError
	}
	return &HTTPError{Code: code, Err: err}
}

func (e *HTTPError) Error() string { return e.Err.Error() }
func (e *HTTPError) Unwrap() error { return e.Err }

func Handler(fn func(w http.ResponseWriter, r *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer closeBody(r.Body)
		if err := fn(w, r); err != nil {
			code := http.StatusInternalServerError
			var he *HTTPError
			if errors.As(err, &he) {
				code = he.Code
				w.WriteHeader(he.Code)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			logging.GetLogger(r).Error("http: error handling request",
				zap.Error(err), zap.Int("status_code", code))
		} else {
			w.WriteHeader(200)
		}
	})
}

// TODO: use [http.MaxBytesReader]
func HandleEcowittPost(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return errors.New("invalid method: " + r.Method)
	}
	if err := r.ParseForm(); err != nil {
		return NewHTTPError(http.StatusBadRequest, err)
	}
	d, err := ParseFormData(r.Form)
	if err != nil {
		return NewHTTPError(http.StatusInternalServerError, err)
	}
	log := logging.GetLogger(r)
	log.Debug("ecowitt post", zap.Any("data", d))
	if err := UpdateMetrics(d); err != nil {
		return NewHTTPError(http.StatusInternalServerError, err)
	}
	if globalConn != nil {
		if err := InsertRow(r.Context(), globalConn, d); err != nil {
			log.Error("insert", zap.Error(err))
			return err
		}
	}
	return nil
}

var globalConn *pgx.Conn

func realMain(log *zap.Logger) error {
	// if _, err := conn.Exec(ctx, createTableStmt); err != nil {
	envOr := func(key, def string) string {
		if s := os.Getenv(key); s != "" {
			return s
		}
		return def
	}
	addr := pflag.String("addr", envOr("ECOWITT_ADDR", ":3002"), "HTTP servder address.")
	dbUser := pflag.String("db-user", envOr("ECOWITT_DB_USER", "REPLACE_ME"), "Postgres user.")
	dbPass := pflag.String("db-password", envOr("ECOWITT_DB_PASSWORD", "REPLACE_ME"), "Postgres password.")
	dbName := pflag.String("db-name", envOr("ECOWITT_DB_NAME", "weather"), "Postgres database.")
	pflag.Parse()

	var signaled atomic.Bool
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGSTOP)
	go func() {
		<-ctx.Done()
		signaled.Store(true)
		log.Error("signaled: stopping now")
		cancel()
	}()

	conn, err := pgx.Connect(ctx, fmt.Sprintf("postgres://%s:%s@localhost:5432/%s",
		*dbUser, *dbPass, *dbName))
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	ping := func(parent context.Context) error {
		ctx, cancel := context.WithTimeout(parent, 5*time.Second)
		defer cancel()
		return conn.Ping(ctx)
	}
	if err := ping(ctx); err != nil {
		log.Error("ping failed", zap.Error(err))
		return err
	}
	globalConn = conn

	if _, err := conn.Exec(ctx, createTableStmt); err != nil {
		log.Error("failed to created tables", zap.Error(err))
		return err
	}

	srv := NewServer(ctx, *addr, Routes())
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		log.Info("signaled: stopping server")
		if err := srv.Shutdown(sctx); err != nil {
			log.Error("server: error during shutdown", zap.Error(err))
		}
	}()

	if err := srv.ListenAndServe(); err != nil {
		// Ignore errors due to normal termination signals
		if !(errors.Is(err, http.ErrServerClosed) && signaled.Load()) {
			return err
		}
	}
	return nil
}

func main() {
	// 3C9AEE4C4663027FD3D386B51441801E
	log := zap.L().Named("main")
	if err := realMain(log); err != nil {
		log.Fatal("server exited with error", zap.Error(err))
	}
}

/*
func GetLiveData() (any, error) {
	res, err := http.Get("http://10.0.1.200/get_livedata_info")
	if err != nil {
		return nil, err
	}
	defer closeBody(res.Body)
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("bad status code: %d", res.StatusCode)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v, nil
}
*/
