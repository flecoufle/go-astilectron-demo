package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/asticode/go-astichartjs"
	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bootstrap"
)

func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// handleMessages handles messages
func handleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "explore":
		// Unmarshal payload
		var path string
		if len(m.Payload) > 0 {
			// Unmarshal payload
			if err = json.Unmarshal(m.Payload, &path); err != nil {
				payload = err.Error()
				return
			}
		}

		// Explore
		if payload, err = explore(path); err != nil {
			payload = err.Error()
			return
		}
	}
	return
}

// Exploration represents the results of an exploration
type Exploration struct {
	Dirs       []Dir              `json:"dirs"`
	Files      *astichartjs.Chart `json:"files,omitempty"`
	FilesCount int                `json:"files_count"`
	FilesSize  string             `json:"files_size"`
	Path       string             `json:"path"`
}

// PayloadDir represents a dir payload
type Dir struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Prepare fileSize
func formatFileSize(filesSize int64) (stringSize string) {
	if filesSize < 1e3 {
		stringSize = strconv.Itoa(int(filesSize)) + "o"
	} else if filesSize < 1e6 {
		stringSize = strconv.FormatFloat(float64(filesSize)/float64(1024), 'f', 0, 64) + "Ko"
	} else if filesSize < 1e9 {
		stringSize = strconv.FormatFloat(float64(filesSize)/float64(1024*1024), 'f', 0, 64) + "Mo"
	} else {
		stringSize = strconv.FormatFloat(float64(filesSize)/float64(1024*1024*1024), 'f', 0, 64) + "Go"
	}
	return
}

var KB = uint64(1024)

func calcDirSize(p string) int64 {
	v, _ := DirSize(p)
	return v
}

// explore explores a path.
// If path is empty, it explores the user's home directory
func explore(path string) (e Exploration, err error) {
	// If no path is provided, use the user's home dir
	if len(path) == 0 {
		var u *user.User
		if u, err = user.Current(); err != nil {
			return
		}
		path = filepath.Join(filepath.Join(u.HomeDir, "Documents"), "Perso")
	}

	// Read dir
	var files []os.FileInfo
	if files, err = ioutil.ReadDir(path); err != nil {
		return
	}

	// Init exploration
	e = Exploration{
		Dirs: []Dir{},
		Path: path,
	}

	// Add previous dir
	if filepath.Dir(path) != path {
		e.Dirs = append(e.Dirs, Dir{
			Name: "..",
			Path: filepath.Dir(path),
		})
	}

	// Loop through files
	var sizes []int
	var sizesMap = make(map[int][]string)
	var filesSize int64
	for _, f := range files {
		if f.IsDir() {
			dirSize := calcDirSize(filepath.Join(path, f.Name()))
			e.Dirs = append(e.Dirs, Dir{
				Name: f.Name() + " (" + formatFileSize(dirSize) + ")",
				Path: filepath.Join(path, f.Name()),
			})
		} else {
			var s = int(f.Size())
			sizes = append(sizes, s)
			sizesMap[s] = append(sizesMap[s], f.Name()+" ("+formatFileSize(f.Size())+")")
			e.FilesCount++
			filesSize += f.Size()
		}
	}

	// Prepare files size
	e.FilesSize = formatFileSize(filesSize)

	// Prepare files chart
	sort.Ints(sizes)
	if len(sizes) > 0 {
		e.Files = &astichartjs.Chart{
			Data: &astichartjs.Data{Datasets: []astichartjs.Dataset{{
				BackgroundColor: []string{
					astichartjs.ChartBackgroundColorYellow,
					astichartjs.ChartBackgroundColorGreen,
					astichartjs.ChartBackgroundColorRed,
					astichartjs.ChartBackgroundColorBlue,
					astichartjs.ChartBackgroundColorPurple,
				},
				BorderColor: []string{
					astichartjs.ChartBorderColorYellow,
					astichartjs.ChartBorderColorGreen,
					astichartjs.ChartBorderColorRed,
					astichartjs.ChartBorderColorBlue,
					astichartjs.ChartBorderColorPurple,
				},
			}}},
			Type: astichartjs.ChartTypePie,
		}
		var sizeOther int
		for i := len(sizes) - 1; i >= 0; i-- {
			for _, l := range sizesMap[sizes[i]] {
				if len(e.Files.Data.Labels) < 4 {
					e.Files.Data.Datasets[0].Data = append(e.Files.Data.Datasets[0].Data, sizes[i])
					e.Files.Data.Labels = append(e.Files.Data.Labels, l)
				} else {
					sizeOther += sizes[i]
				}
			}
		}
		if sizeOther > 0 {
			e.Files.Data.Datasets[0].Data = append(e.Files.Data.Datasets[0].Data, sizeOther)
			e.Files.Data.Labels = append(e.Files.Data.Labels, "other")
		}
	}
	return
}
