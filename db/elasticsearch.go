package db

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/elastic/go-elasticsearch/v7/estransport"
	"github.com/natefinch/lumberjack"
)

type ElasticsearchConfig struct {
	Hosts    []string
	Username string // Username for HTTP Basic Authentication.
	Password string
	Log      *struct {
		Path       string
		MaxSize    int  // 在进行切割之前，日志文件的最大大小（以MB为单位）
		MaxBackups int  // 保留旧文件的最大个数
		MaxAge     int  // 保留旧文件的最大天数
		Compress   bool // 是否压缩/归档旧文件
	}
}

type Elasticsearch struct {
	cli *elasticsearch7.Client
}

func NewElasticsearch(cfg *ElasticsearchConfig) (*Elasticsearch, error) {
	if len(cfg.Hosts) == 0 {
		cfg.Hosts = []string{"http://127.0.0.1:19200"}
	}
	var log estransport.Logger
	if cfg.Log != nil && len(cfg.Log.Path) != 0 {
		lcfg := cfg.Log
		lumberJackLogger := &lumberjack.Logger{
			Filename:   lcfg.Path,
			MaxSize:    lcfg.MaxSize,    //在进行切割之前，日志文件的最大大小（以MB为单位）
			MaxBackups: lcfg.MaxBackups, //保留旧文件的最大个数
			MaxAge:     lcfg.MaxAge,     //保留旧文件的最大天数
			Compress:   lcfg.Compress,   //是否压缩/归档旧文件
		}
		log = &estransport.TextLogger{Output: lumberJackLogger, EnableRequestBody: true, EnableResponseBody: true}
	} else {
		log = &estransport.ColorLogger{Output: os.Stdout, EnableRequestBody: true, EnableResponseBody: true}
	}
	cli, err := elasticsearch7.NewClient(elasticsearch7.Config{
		Addresses: cfg.Hosts,
		Username:  cfg.Username,
		Password:  cfg.Password,
		Logger:    log,
	})
	return &Elasticsearch{cli: cli}, err
}

func (e *Elasticsearch) Client() *elasticsearch7.Client {
	return e.cli
}

func (e *Elasticsearch) Index(iname string) error {
	res, err := e.cli.Indices.Exists([]string{iname})
	if err != nil {
		return err
	}
	if res.StatusCode == 200 {
		return nil
	}
	res, err = e.cli.Indices.Create(iname)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return errors.New(res.String())
	}
	if res.StatusCode == 200 {
		return nil
	}
	return nil
}

func (e *Elasticsearch) ExistsID(index, id string) error {
	res, err := e.cli.Exists(index, id)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return errors.New(res.String())
	}
	if res.StatusCode == 200 {
		return nil
	}
	return errors.New(res.String())
}

func (e *Elasticsearch) Insert(index, ty, id string, item interface{}) error {
	payload, err := json.Marshal(item)
	if err != nil {
		return err
	}
	res, err := esapi.CreateRequest{
		Index:        index,
		DocumentType: ty,
		DocumentID:   id,
		Body:         bytes.NewReader(payload),
	}.Do(context.Background(), e.cli)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err := elasticError(res); err != nil {
		return err
	}
	return nil
}

func (e *Elasticsearch) Search(index, q string, after []string, offset, limit int, orders []string, out any) error {
	res, err := e.cli.Search(
		// e.cli.Search.WithDocumentType(v ...string)
		e.cli.Search.WithIndex(index),
		e.cli.Search.WithBody(e.buildQuery(q, after...)),
		e.cli.Search.WithFrom(offset),
		e.cli.Search.WithSize(limit),
		e.cli.Search.WithSort(orders...),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err := elasticError(res); err != nil {
		return err
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func (e *Elasticsearch) buildQuery(query string, after ...string) io.Reader {
	var b strings.Builder

	b.WriteString("{\n")

	if query == "" {
		b.WriteString("")
	} else {
		b.WriteString(query)
	}

	if len(after) > 0 && after[0] != "" && after[0] != "null" {
		b.WriteString(",\n")
		b.WriteString(fmt.Sprintf(`	"search_after": %s`, after))
	}

	b.WriteString("\n}")

	// fmt.Printf("%s\n", b.String())
	return strings.NewReader(b.String())
}

func elasticError(res *esapi.Response) error {
	if res.IsError() {
		if res.IsError() {
			var e map[string]interface{}
			if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
				return err
			}
			return fmt.Errorf("[%s] %s: %s", res.Status(), e["error"].(map[string]interface{})["type"], e["error"].(map[string]interface{})["reason"])
		}
	}
	return nil
}
