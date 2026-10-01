// Package e2etest runs a disposable local test process. Production server must not import it.
package e2etest

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/httpapi"
	"github.com/yyl1212/math_master/backend/internal/store"
)

type Config struct{ TestDatabaseURL, APIAddr, ControlAddr, StateFile string }
type runtimeState struct {
	APIURL      string `json:"apiURL"`
	ControlURL  string `json:"controlURL"`
	Token       string `json:"token"`
	Database    string `json:"database"`
	AssetSHA    string `json:"assetSha"`
	KnowledgeID string `json:"knowledgeId"`
	PathID      string `json:"pathId"`
}

var testName = regexp.MustCompile(`^/math_master_test_[a-z0-9_]+$`)

func localAddress(addr string) bool {
	host, port, e := net.SplitHostPort(addr)
	ip := net.ParseIP(host)
	return e == nil && ip != nil && ip.IsLoopback() && port != ""
}
func validate(c Config) (*url.URL, error) {
	u, e := url.Parse(c.TestDatabaseURL)
	if e != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || !testName.MatchString(u.Path) || u.Host == "" || !localAddress(c.APIAddr) || !localAddress(c.ControlAddr) || !strings.HasSuffix(c.StateFile, ".local.json") {
		return nil, errors.New("unsafe harness configuration")
	}
	return u, nil
}
func rootDir() (string, error) {
	dir, e := os.Getwd()
	if e != nil {
		return "", errors.New("harness workspace unavailable")
	}
	for {
		if _, e := os.Stat(filepath.Join(dir, "content/catalogue/domains.json")); e == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("harness workspace unavailable")
		}
		dir = parent
	}
}
func loadFixture(root string, long bool) (content.ValidatedPackage, error) {
	f, e := os.Open(filepath.Join(root, "content/catalogue/domains.json"))
	if e != nil {
		return content.ValidatedPackage{}, errors.New("fixture unavailable")
	}
	c, e := content.DecodeCatalogue(f)
	f.Close()
	if e != nil {
		return content.ValidatedPackage{}, errors.New("fixture catalogue invalid")
	}
	f, e = os.Open(filepath.Join(root, "content/packages/elementary-fractions.v1.json"))
	if e != nil {
		return content.ValidatedPackage{}, errors.New("fixture unavailable")
	}
	p, e := content.DecodePackage(f)
	f.Close()
	if e != nil {
		return content.ValidatedPackage{}, errors.New("fixture package invalid")
	}
	if long {
		p.ID = "e2e-long-reading"
		for i := range p.Knowledge {
			k := &p.Knowledge[i]
			k.Version = 2
			k.Title += " — Understanding exact assumptions and connected mathematical ideas across a carefully ordered learning journey"
			k.TitleZh += "：理解条件与关联知识，沿着清晰的学习路径循序渐进"
			for j := range k.Relations {
				k.Relations[j].Target.Version = 2
			}
		}
		for i := range p.Units {
			p.Units[i].Version = 2
			p.Units[i].Knowledge.Version = 2
		}
		for i := range p.Assets {
			p.Assets[i].Knowledge.Version = 2
		}
		for i := range p.Paths {
			p.Paths[i].Version = 2
			for j := range p.Paths[i].Nodes {
				p.Paths[i].Nodes[j].Version = 2
			}
		}
		p.Units[0].Angles = append(p.Units[0].Angles, content.Angle{Kind: "formal", Body: "Test-only long display and inline formulas.\n\n$$" + strings.Repeat("x+", 90) + "x$$\n\n$" + strings.Repeat("x+", 90) + "x$"})
	}
	// Only the disposable database publishes these fixtures; this does not constitute mathematical review.
	p.Units[0].Angles = append(p.Units[0].Angles, content.Angle{Kind: "formal", Body: `For nonzero b and k, $\frac{a}{b}=\frac{ka}{kb}$.`})
	v, report := content.ValidateAndSeal(c, p, filepath.Join(root, "content/assets"))
	if len(report.Errors) > 0 {
		return content.ValidatedPackage{}, errors.New("fixture validation failed")
	}
	return v, nil
}

// Run owns only its freshly generated database and exclusively created local state file.
func Run(ctx context.Context, c Config) (result error) {
	u, e := validate(c)
	if e != nil {
		return e
	}
	root, e := rootDir()
	if e != nil {
		return e
	}
	normal, e := loadFixture(root, false)
	if e != nil {
		return e
	}
	long, e := loadFixture(root, true)
	if e != nil {
		return e
	}
	random := make([]byte, 32)
	if _, e = rand.Read(random); e != nil {
		return errors.New("harness randomness unavailable")
	}
	name := "math_master_test_" + hex.EncodeToString(random[:8])
	token := hex.EncodeToString(random[8:])
	adminURL := *u
	adminURL.Path = "/postgres"
	admin, e := sql.Open("pgx", adminURL.String())
	if e != nil {
		return errors.New("harness database configuration failed")
	}
	defer admin.Close()
	setup, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if _, e = admin.ExecContext(setup, `CREATE DATABASE "`+name+`"`); e != nil {
		return errors.New("harness isolated database creation failed")
	}
	var db *sql.DB
	stateCreated := false
	defer func() {
		if db != nil {
			db.Close()
		}
		cleanup, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		if _, err := admin.ExecContext(cleanup, `DROP DATABASE "`+name+`" WITH (FORCE)`); err != nil {
			result = errors.Join(result, errors.New("harness database cleanup failed; retain state for guarded recovery"))
			return
		}
		if stateCreated {
			if err := os.Remove(c.StateFile); err != nil && !os.IsNotExist(err) {
				result = errors.Join(result, errors.New("harness state cleanup failed"))
			}
		}
	}()
	u.Path = "/" + name
	db, e = sql.Open("pgx", u.String())
	if e != nil {
		return errors.New("harness isolated connection failed")
	}
	db.SetMaxOpenConns(6)
	if e = store.Up(setup, db, filepath.Join(root, "db/migrations")); e != nil {
		return errors.New("harness migration failed")
	}
	s := store.New(db)
	var mu sync.RWMutex
	unavailable := false
	change := func(ctx context.Context, scene string) error {
		mu.Lock()
		defer mu.Unlock()
		ctx, stop := context.WithTimeout(ctx, 3*time.Second)
		defer stop()
		if scene == "unavailable" {
			unavailable = true
			return nil
		}
		switch scene {
		case "empty", "draft", "published", "long":
			if _, e := db.ExecContext(ctx, "TRUNCATE catalogue_versions, knowledge, path_versions, assets CASCADE"); e != nil {
				return errors.New("scene reset failed")
			}
			if scene != "empty" {
				v := normal
				if scene == "long" {
					v = long
				}
				if _, e := s.ImportDraft(ctx, v); e != nil {
					return errors.New("scene import failed")
				}
				if scene == "published" || scene == "long" {
					if _, e := db.ExecContext(ctx, "UPDATE publication_snapshots SET status='published'; INSERT INTO publication_heads SELECT true,id FROM publication_snapshots LIMIT 1"); e != nil {
						return errors.New("test publication failed")
					}
				}
			}
		case "withdraw-asset", "withdraw-unit", "withdraw-knowledge", "withdraw-prerequisite":
			kind, id := "asset", normal.Package().Assets[0].ID
			switch scene {
			case "withdraw-unit":
				kind, id = "unit", normal.Package().Units[0].ID
			case "withdraw-knowledge":
				kind, id = "knowledge", "equivalent-fractions"
			case "withdraw-prerequisite":
				kind, id = "knowledge", "numbers"
			}
			if _, e := db.ExecContext(ctx, "UPDATE publication_members SET availability='withdrawn' WHERE snapshot_id=(SELECT snapshot_id FROM publication_heads) AND kind=$1 AND id=$2", kind, id); e != nil {
				return errors.New("test withdrawal failed")
			}
		case "withdraw-snapshot":
			if _, e := db.ExecContext(ctx, "UPDATE publication_snapshots SET status='withdrawn' WHERE id=(SELECT snapshot_id FROM publication_heads)"); e != nil {
				return errors.New("test withdrawal failed")
			}
		default:
			return errors.New("unknown scene")
		}
		unavailable = false
		return nil
	}
	if e = change(setup, "draft"); e != nil {
		return e
	}
	apiListener, e := net.Listen("tcp", c.APIAddr)
	if e != nil {
		return errors.New("harness API bind failed")
	}
	defer apiListener.Close()
	controlListener, e := net.Listen("tcp", c.ControlAddr)
	if e != nil {
		return errors.New("harness control bind failed")
	}
	defer controlListener.Close()
	actual := httpapi.NewHandler(s, db)
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		if unavailable {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(503)
			w.Write([]byte(`{"error":{"code":"UNAVAILABLE","message":"Service unavailable"}}`))
			return
		}
		actual.ServeHTTP(w, r)
	})
	control := http.NewServeMux()
	control.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	control.HandleFunc("POST /scene/{scene}", func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
			http.Error(w, "Unauthorized", 401)
			return
		}
		if e := change(r.Context(), r.PathValue("scene")); e != nil {
			http.Error(w, "Scene change failed", 400)
			return
		}
		w.WriteHeader(204)
	})
	state := runtimeState{APIURL: "http://" + apiListener.Addr().String(), ControlURL: "http://" + controlListener.Addr().String(), Token: token, Database: name, AssetSHA: normal.Package().Assets[0].SHA256, KnowledgeID: "equivalent-fractions", PathID: normal.Package().Paths[0].ID}
	f, e := os.OpenFile(c.StateFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return errors.New("harness requires a new local state file")
	}
	stateCreated = true
	e = json.NewEncoder(f).Encode(state)
	closeErr := f.Close()
	if e != nil || closeErr != nil {
		return errors.New("harness state write failed")
	}
	server := func(h http.Handler) *http.Server {
		return &http.Server{Handler: h, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	}
	api, ctl := server(apiHandler), server(control)
	defer api.Close()
	defer ctl.Close()
	failed := make(chan error, 2)
	go func() { failed <- api.Serve(apiListener) }()
	go func() { failed <- ctl.Serve(controlListener) }()
	select {
	case <-ctx.Done():
	case <-failed:
		return errors.New("harness HTTP server failed")
	}
	// Stop serving before closing connections or dropping the database.
	shutdown, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	if api.Shutdown(shutdown) != nil {
		api.Close()
	}
	if ctl.Shutdown(shutdown) != nil {
		ctl.Close()
	}
	return nil
}
