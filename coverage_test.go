package main

import (
	"bytes"
	"container/list"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

var modules = []mod{
	mod{name: "acodec", decl: buildRegex("ALLEGRO_ACODEC_FUNC")},
	//mod{name: "audio", decl: buildRegex("ALLEGRO_KCM_AUDIO_FUNC")},
	mod{name: "color", decl: buildRegex("ALLEGRO_COLOR_FUNC")},
	mod{name: "dialog", decl: buildRegex("ALLEGRO_DIALOG_FUNC"), header: "native_dialog"},
	mod{name: "font", decl: buildRegex("ALLEGRO_FONT_FUNC")},
	mod{name: "image", decl: buildRegex("ALLEGRO_IIO_FUNC")},
	mod{name: "memfile", decl: buildRegex("ALLEGRO_MEMFILE_FUNC")},
	mod{name: "physfs", decl: buildRegex("ALLEGRO_PHYSFS_FUNC")},
	mod{name: "ttf", decl: buildRegex("ALLEGRO_TTF_FUNC"), path: "font/ttf"},
}

// Ignore functions that I either don't know how to implement,
// are redundant, or are based on Allegro features that Go
// already provides, like file I/O and fixed point math.
var blacklist = map[string]bool{
	"al_register_bitmap_loader":      true,
	"al_register_bitmap_saver":       true,
	"al_register_bitmap_loader_f":    true,
	"al_register_bitmap_saver_f":     true,
	"al_load_bitmap_f":               true,
	"al_save_bitmap_f":               true,
	"al_load_config_file_f":          true,
	"al_save_config_file_f":          true,
	"al_run_main":                    true,
	"al_toggle_display_flag":         true, // deprecated
	"al_fopen_interface":             true,
	"al_create_file_handle":          true,
	"al_fungetc":                     true,
	"al_fgetc":                       true,
	"al_fputc":                       true,
	"al_fread16le":                   true,
	"al_fread16be":                   true,
	"al_fwrite16le":                  true,
	"al_fwrite16be":                  true,
	"al_fread32le":                   true,
	"al_fread32be":                   true,
	"al_fwrite32le":                  true,
	"al_fwrite32be":                  true,
	"al_fgets":                       true,
	"al_fget_ustr":                   true,
	"al_fputs":                       true,
	"al_fopen_fd":                    true,
	"al_get_new_file_interface":      true,
	"al_set_standard_file_interface": true,
	"al_get_file_userdata":           true,
	"al_fixsqrt":                     true,
	"al_fixhypot":                    true,
	"al_fixatan":                     true,
	"al_fixatan2":                    true,
	"al_destroy_fs_entry":            true,
	"al_get_current_directory":       true,
	"al_change_directory":            true,
	"al_get_fs_interface":            true,
	"al_set_fs_interface":            true,
	"al_set_standard_fs_interface":   true,
	"al_create_path":                 true,
	"al_create_path_for_directory":   true,
	"al_clone_path":                  true,
	"al_get_path_num_components":     true,
	"al_get_path_component":          true,
	"al_replace_path_component":      true,
	"al_remove_path_component":       true,
	"al_insert_path_component":       true,
	"al_get_path_tail":               true,
	"al_drop_path_tail":              true,
	"al_append_path_component":       true,
	"al_join_paths":                  true,
	"al_rebase_path":                 true,
	"al_path_cstr":                   true,
	"al_destroy_path":                true,
	"al_set_path_drive":              true,
	"al_get_path_drive":              true,
	"al_set_path_filename":           true,
	"al_get_path_filename":           true,
	"al_get_path_extension":          true,
	"al_set_path_extension":          true,
	"al_get_path_basename":           true,
	"al_make_path_canonical":         true,
	"al_install_system":              true, // taken care of automatically
	"al_uninstall_system":            true,
	"al_is_system_installed":         true,
	"al_get_system_driver":           true, // why was this reported?
	"al_get_standard_path":           true, // could be useful, if we can convert ALLEGRO_PATH structs to strings
	"al_start_thread":                true,
	"al_join_thread":                 true,
	"al_set_thread_should_stop":      true,
	"al_get_thread_should_stop":      true,
	"al_destroy_thread":              true,
	"al_run_detached_thread":         true,
	"al_create_mutex":                true,
	"al_create_mutex_recursive":      true,
	"al_lock_mutex":                  true,
	"al_unlock_mutex":                true,
	"al_destroy_mutex":               true,
	"al_create_cond":                 true,
	"al_destroy_cond":                true,
	"al_wait_cond":                   true,
	"al_broadcast_cond":              true,
	"al_signal_cond":                 true,
	"al_ustr_new":                    true,
	"al_ustr_new_from_buffer":        true,
	"al_ustr_free":                   true,
	"al_cstr":                        true,
	"al_ustr_to_buffer":              true,
	"al_cstr_dup":                    true,
	"al_ustr_dup":                    true,
	"al_ustr_empty_string":           true,
	"al_ref_cstr":                    true,
	"al_ustr_size":                   true,
	"al_ustr_length":                 true,
	"al_ustr_offset":                 true,
	"al_ustr_next":                   true,
	"al_ustr_prev":                   true,
	"al_ustr_get":                    true,
	"al_ustr_get_next":               true,
	"al_ustr_prev_get":               true,
	"al_ustr_insert_chr":             true,
	"al_ustr_append":                 true,
	"al_ustr_append_cstr":            true,
	"al_ustr_append_chr":             true,
	"al_ustr_remove_chr":             true,
	"al_ustr_truncate":               true,
	"al_ustr_ltrim_ws":               true,
	"al_ustr_rtrim_ws":               true,
	"al_ustr_trim_ws":                true,
	"al_ustr_assign":                 true,
	"al_ustr_assign_cstr":            true,
	"al_ustr_set_chr":                true,
	"al_ustr_equal":                  true,
	"al_ustr_compare":                true,
	"al_ustr_has_prefix_cstr":        true,
	"al_utf8_width":                  true,
	"al_utf8_encode":                 true,
	"al_ustr_new_from_utf16":         true,
	"al_ustr_size_utf16":             true,
	"al_ustr_encode_utf16":           true,
	"al_utf16_width":                 true,
	"al_utf16_encode":                true,
	"al_get_errno":                   true,
	"al_set_errno":                   true,
	"al_register_font_loader":        true,
	"al_get_ustr_width":              true,
	"al_draw_ustr":                   true,
	"al_draw_justified_ustr":         true,
	"al_get_ustr_dimensions":         true,
	"al_load_ttf_font_f":             true,
	"al_load_ttf_font_stretch_f":     true,
}

type mod struct {
	name, path, header string
	decl      *decl
}

func (m *mod) Header() string {
	if m.header != "" {
		return "allegro_" + m.header + ".h"
	} else {
		return "allegro_" + m.name + ".h"
	}
}

func (m *mod) Path() string {
	if m.path != "" {
		return m.path
	} else {
		return m.name
	}
}

type decl struct {
	macro string
	regex *regexp.Regexp
}

// regexes for various function macros
var alFunc = buildRegex("AL_FUNC")

func buildRegex(macro string) *decl {
	return &decl{macro, regexp.MustCompile(macro + `\((?P<type>.*), (?P<name>.*), \((?P<params>.*)\)\)`)}
}

func getSource(packageRoot string) ([]byte, error) {
	var buf bytes.Buffer
	err := filepath.Walk(packageRoot, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && path != packageRoot {
			return filepath.SkipDir
		} else if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}
		data, err2 := ioutil.ReadFile(path)
		if err2 != nil {
			return fmt.Errorf("can't read Go source file \"%s\": %s", path, err2.Error())
		}
		buf.Write(data)
		return nil
	})
	if err != nil {
		panic(err)
	}
	return buf.Bytes(), err
}

type missingFunc struct {
	Name   string
	Type   string
	Params string
	Header string
	Module string
}

func scanHeaders(packageRoot string, missingFuncs chan *missingFunc, errs chan error) {
	var (
		source     []byte
		sourceErr  error
		headerRoot = filepath.Join("/", "usr", "include", "allegro5")
	)

	defer func() {
		close(missingFuncs)
		close(errs)
	}()

	// First walk the full root, looking for standard allegro functions.
	source, sourceErr = getSource(packageRoot)
	if sourceErr != nil {
		errs <- sourceErr
		return
	}
	filepath.Walk(headerRoot, func(header string, info os.FileInfo, err error) error {
		if info.IsDir() && info.Name() == "internal" {
			return filepath.SkipDir
		} else if info.IsDir() || !strings.HasSuffix(info.Name(), ".h") {
			return nil
		}
		data, err2 := ioutil.ReadFile(header)
		if err2 != nil {
			errs <- err2
			return nil
		}
		findMissingFuncs(data, source, header, alFunc, "", missingFuncs)
		return nil
	})

	// Now iterate through all known modules.
	for _, m := range modules {
		root := filepath.Join(packageRoot, m.Path())
		header := filepath.Join(headerRoot, m.Header())
		if _, err := os.Stat(header); os.IsNotExist(err) {
			errs <- fmt.Errorf("Module header not found at '%s'", header)
			continue
		}
		if info, err := os.Stat(root); os.IsNotExist(err) || !info.IsDir() {
			errs <- fmt.Errorf("Source not found at '%s'", root)
			continue
		}
		data, err := ioutil.ReadFile(header)
		if err != nil {
			errs <- err
			continue
		}
		source, sourceErr = getSource(root)
		if sourceErr != nil {
			errs <- sourceErr
			return
		}
		findMissingFuncs(data, source, header, m.decl, m.name, missingFuncs)
	}
}

// findMissingFuncs() is a customized iteration method used to ensure that multi-line function declarations are found.
func findMissingFuncs(data, source []byte, header string, d *decl, modName string, missingFuncs chan *missingFunc) {
	ch := make(chan string)
	go func() {
		var buf bytes.Buffer
		lines := strings.Split(string(data), "\n")
		for i := 0; i<len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			buf.WriteString(line)
			if strings.HasPrefix(line, d.macro) {
				for !strings.HasSuffix(line, ";") {
					i++
					line = strings.TrimSpace(lines[i])
					buf.WriteString(line)
				}
			}
			ch <- buf.String()
			buf.Reset()
		}
		close(ch)
	}()
	for line := range ch {
		vals := d.regex.FindStringSubmatch(line)
		if vals == nil {
			// no match
			continue
		}
		name := strings.TrimSpace(vals[2])
		if strings.HasPrefix(name, "_") {
			// function names starting with an underscore are private
			continue
		}
		typ := strings.TrimSpace(vals[1])
		params := strings.TrimSpace(vals[3])
		if !bytes.Contains(source, []byte("C."+name)) {
			missingFuncs <- &missingFunc{Name: name, Type: typ, Params: params, Module: modName, Header: header}
		}
	}
}

func TestCoverage(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err.Error())
	}
	packageRoot := filepath.Join(cwd, "allegro")

	missingFuncs := make(chan *missingFunc)
	errs := make(chan error)
	go scanHeaders(packageRoot, missingFuncs, errs)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		for f := range missingFuncs {
			if blacklist[f.Name] {
				continue
			}
			if f.Module == "" {
				t.Errorf("Missing allegro function '%s' in file '%s'", f.Name, f.Header)
			} else {
				t.Errorf("Module '%s' missing function '%s' [%s %s(%s)]", f.Module, f.Name, f.Type, f.Name, f.Params)
			}
		}
		wg.Done()
	}()

	go func() {
		errorList := list.New()
		for err := range errs {
			errorList.PushBack(err)
		}
		for e := errorList.Front(); e != nil; e = e.Next() {
			t.Error("Error: " + e.Value.(error).Error())
		}
		wg.Done()
	}()

	wg.Wait()
}
