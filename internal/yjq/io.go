package yjq

import (
	"encoding/json"
	"io"
	"os"
	"path"

	yaml "gopkg.in/yaml.v2"
)

func CopyFile(filename string, w io.Writer) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	switch path.Ext(filename) {
	case ".yaml", ".yml":
		return CopyYAMLToJSON(w, f)
	default:
		_, err := io.Copy(w, f)
		return err
	}
}

func CopyYAMLToJSON(w io.Writer, r io.Reader) (err error) {
	dec := yaml.NewDecoder(r)
	enc := json.NewEncoder(w)
	for {
		v := Value{}
		if err = dec.Decode(&v); err != nil {
			if err == io.EOF {
				return nil
			}
			return
		}
		if err = enc.Encode(&v); err != nil {
			return err
		}
	}
}

func CopyJSONToYAML(w io.Writer, r io.Reader) (n int64, err error) {
	dec := json.NewDecoder(r)
	var (
		data []byte
		nn   int
	)

	for i := 0; true; i++ {
		v := Value{}
		if err = dec.Decode(&v); err != nil {
			if err == io.EOF {
				return 0, nil
			}
			return
		}
		if i > 0 {
			nn, err = w.Write([]byte{'-', '-', '-', '\n'})
			n += int64(nn)
			if err != nil {
				return
			}
		}
		if v.Type == Null {
			nn, err = w.Write([]byte{'\n'})
			n += int64(nn)
			if err != nil {
				return
			}
			continue
		}
		data, err = yaml.Marshal(&v)
		if err != nil {
			return
		}
		nn, err = w.Write(data)
		n += int64(nn)
		if err != nil {
			return
		}
	}
	return
}
