package apki

import (
	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	_ "github.com/cayleygraph/cayley/graph/kv/badger"
	_ "github.com/cayleygraph/cayley/graph/kv/bolt"
	"github.com/cayleygraph/quad"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/wenerme/tools/pkg/apk"
	"os"
)

type Graph struct {
	Store *cayley.Handle
}

var g *Graph

func GetDependencies(p string) []string {
	return nil
}

func GetGraph() (*Graph, error) {
	if g != nil {
		return g, nil
	}
	path := "/tmp/apki.graph.badger"
	v := &Graph{}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := graph.InitQuadStore("badger", path, nil); err != nil {
			return nil, errors.Wrap(err, "Init Quad")
		}
	} else if err != nil {
		return nil, errors.Wrap(err, "check exist")
	}
	store, err := cayley.NewGraph("badger", path, nil)
	if err != nil {
		return nil, err
	}
	v.Store = store

	{
		v.Store = store

		if v, err := store.Stats(nil, false); err != nil {
			return nil, err
		} else if v.Nodes.Size == 0 {
			logrus.WithField("state", "start").Info("index graph")
			m := apk.Mirror("https://mirrors.aliyun.com/alpine")
			idx, err := m.Repo("v3.12", "main", "x86_64").Index()
			if err != nil {
				return nil, err
			}
			if err := IndexPackageGraph(store, idx); err != nil {
				return nil, err
			}
			logrus.WithField("state", "end").Info("index graph")
		} else {
			logrus.WithField("stats", v).Info("graph already init")
		}
	}
	g = v
	return g, nil
}

func IndexPackageGraph(store *cayley.Handle, idx apk.Index) error {
	for _, v := range idx {
		for _, s := range v.Provides {
			dep := apk.ParseDependency(s)
			err := store.AddQuad(quad.Make(v.Name, "provide", dep.Name, s))
			if err != nil {
				return err
			}
		}
		for _, s := range v.Depends {
			dep := apk.ParseDependency(s)
			err := store.AddQuad(quad.Make(v.Name, "depend", dep.Name, s))
			if err != nil {
				return err
			}
		}
		for _, s := range v.InstallIf {
			dep := apk.ParseDependency(s)
			err := store.AddQuad(quad.Make(v.Name, "install-if", dep.Name, s))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
