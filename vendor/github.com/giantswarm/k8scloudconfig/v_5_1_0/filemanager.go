package v_5_1_0

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"text/template"

	"github.com/giantswarm/microerror"

	ignition "github.com/giantswarm/k8scloudconfig/ignition/v_2_2_0"
)

const (
	version           = "v_5_1_0"
	filesDir          = "files"
	additionExtension = ".yaml"
)

// Files is map[string]string (k: filename, v: contents) for files that are fetched from disk
// and then filled with data.
type Files map[string]string

// RenderFiles walks over filesdir and parses all regular files with
// text/template. Parsed templates are then rendered with ctx, base64 encoded
// and added to returned Files.
//
// filesdir must not contain any other files than templates that can be parsed
// with text/template.
func RenderFiles(filesdir string, ctx interface{}) (Files, error) {
	files := Files{}

	err := filepath.Walk(filesdir, func(path string, f os.FileInfo, err error) error {
		if f.Mode().IsRegular() {
			tmpl, err := template.ParseFiles(path)
			if err != nil {
				return microerror.Maskf(err, "failed to parse file %#q", path)
			}
			var data bytes.Buffer
			tmpl.Execute(&data, ctx)

			relativePath, err := filepath.Rel(filesdir, path)
			if err != nil {
				return microerror.Mask(err)
			}
			files[relativePath] = base64.StdEncoding.EncodeToString(data.Bytes())
		}
		return nil
	})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return files, nil
}

// GetIgnitionPath returns path for the ignition assets based on
// base ignition directory and package subdirectory with assets.
func GetIgnitionPath(ignitionDir string) string {
	return filepath.Join(ignitionDir, version, filesDir)
}

// GetPackagePath returns top package path for the current runtime file.
// For example, for /go/src/k8scloudconfig/v_4_1_0/file.go function
// returns /go/src/k8scloudconfig.
// This function used only in tests for retrieving ignition assets in runtime.
func GetPackagePath() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", microerror.New("failed to retrieve runtime information")
	}

	return filepath.Dir(filepath.Dir(filename)), nil
}

// GetIgnitionAdditions reads .yaml files from specified folder
// and returns list of base64 strings of valid ignition configurations.
func GetIgnitionAdditions(additionPath string) ([]string, error) {
	var ignitionAdditions []string

	var additionPaths []string
	err := filepath.Walk(additionPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			r, err := regexp.MatchString(additionExtension, info.Name())
			if err == nil && r {
				additionPaths = append(additionPaths, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for _, ext := range additionPaths {
		data, err := ioutil.ReadFile(ext)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		ignitionJSON, err := ignition.ConvertTemplatetoJSON(data)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		ignitionAdditions = append(ignitionAdditions, base64.StdEncoding.EncodeToString(ignitionJSON))
	}

	return ignitionAdditions, nil
}
