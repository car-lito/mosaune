package acquisition

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// WireFormat maps the public acquisition vocabulary to the service's JSON.
// No response, path, credential or configured destination enters the module.
type WireFormat struct {
	Version  int                          `json:"version"`
	Fields   map[string]map[string]string `json:"fields"`
	Commands map[string]string            `json:"commands"`
}

func wireVocabulary(connector string) map[string][]string {
	shapes := map[string][]string{
		"status": {"appName"}, "profile": {"id", "name"}, "root": {"id", "path"},
		"resource": {"id", "externalId", "title", "path", "hasFile", "monitored", "qualityProfileId", "rootFolderPath", "addOptions"},
		"file":     {"id", "relativePath"}, "queue": {"records", "totalRecords"},
		"row": {"resourceId", "episodeId", "seasonNumber", "size", "sizeleft", "trackedDownloadStatus"},
	}
	if connector == "radarr" {
		shapes["resource"] = append(shapes["resource"], "minimumAvailability", "movieFile")
		shapes["addOptions"] = []string{"searchForMovie"}
		shapes["search"] = []string{"name", "movieIds"}
	} else {
		shapes["resource"] = append(shapes["resource"], "seasons", "seasonFolder", "seriesType", "monitorNewItems")
		shapes["season"] = []string{"seasonNumber", "monitored"}
		shapes["episode"] = []string{"id", "seasonNumber", "episodeNumber", "hasFile", "episodeFileId"}
		shapes["addOptions"] = []string{"monitor", "searchForMissingEpisodes"}
		shapes["search"] = []string{"name", "seriesId", "seasonNumber"}
		shapes["monitor"] = []string{"episodeIds", "monitored"}
	}
	return shapes
}
func (f WireFormat) Validate(connector string) error {
	bad := errors.New("invalid acquisition JSON format")
	if (connector != "radarr" && connector != "sonarr") || f.Version != 1 {
		return bad
	}
	shapes := wireVocabulary(connector)
	if len(shapes) != len(f.Fields) {
		return bad
	}
	for shape, names := range shapes {
		fields := f.Fields[shape]
		if len(fields) != len(names) {
			return bad
		}
		seen := map[string]bool{}
		for _, name := range names {
			wire := fields[name]
			if len(wire) > 64 || !safeSegment(wire) || seen[strings.ToLower(wire)] {
				return bad
			}
			seen[strings.ToLower(wire)] = true
		}
	}
	names := []string{"movie"}
	if connector == "sonarr" {
		names = []string{"series", "season"}
	}
	if len(f.Commands) != len(names) {
		return bad
	}
	seen := map[string]bool{}
	for _, name := range names {
		value := f.Commands[name]
		if !safeSegment(value) || len(value) > 64 || seen[value] {
			return bad
		}
		seen[value] = true
	}
	return nil
}

func wireShape(operation string, encode bool) (string, bool) {
	if encode {
		switch operation {
		case "create", "update":
			return "resource", false
		case "search", "monitor":
			return operation, false
		}
		return "", false
	}
	switch operation {
	case "status", "queue":
		return operation, false
	case "profiles":
		return "profile", true
	case "roots":
		return "root", true
	case "find", "lookup":
		return "resource", true
	case "create":
		return "resource", false
	case "episodes":
		return "episode", true
	case "episodefiles":
		return "file", true
	}
	return "", false
}
func wireChild(shape, field string) (string, bool) {
	if shape == "resource" {
		switch field {
		case "seasons":
			return "season", true
		case "movieFile":
			return "file", false
		case "addOptions":
			return "addOptions", false
		}
	}
	if shape == "queue" && field == "records" {
		return "row", true
	}
	return "", false
}

// Decode JSON without rounding integers or accepting ambiguous duplicate keys.
// Bounds apply also to unknown fields, which must survive a later PUT verbatim
// in value (their formatting and key order need not survive).
func readWireJSON(d *json.Decoder, depth int, budget *int) (any, error) {
	*budget--
	if depth > 48 || *budget < 0 {
		return nil, errors.New("JSON limit")
	}
	token, err := d.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return token, nil
	}
	switch delim {
	case '{':
		result := map[string]any{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return nil, err
			}
			s, ok := key.(string)
			if !ok {
				return nil, errors.New("JSON key")
			}
			if _, exists := result[s]; exists {
				return nil, errors.New("duplicate JSON key")
			}
			value, err := readWireJSON(d, depth+1, budget)
			if err != nil {
				return nil, err
			}
			result[s] = value
		}
		_, err = d.Token()
		return result, err
	case '[':
		result := []any{}
		for d.More() {
			value, err := readWireJSON(d, depth+1, budget)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		_, err = d.Token()
		return result, err
	}
	return nil, errors.New("unexpected JSON delimiter")
}

func (f WireFormat) transform(shape string, array bool, value any, encode bool) (any, error) {
	if value == nil {
		return nil, nil
	}
	bad := errors.New("ambiguous or incompatible acquisition data")
	if array {
		list, ok := value.([]any)
		if !ok {
			return nil, bad
		}
		for i, item := range list {
			next, err := f.transform(shape, false, item, encode)
			if err != nil {
				return nil, err
			}
			list[i] = next
		}
		return list, nil
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return nil, bad
	}
	fields, ok := f.Fields[shape]
	if !ok {
		return nil, bad
	}
	mapping := fields
	if !encode {
		mapping = map[string]string{}
		for name, wire := range fields {
			mapping[wire] = name
		}
	}
	out := make(map[string]any, len(obj))
	seen := map[string]bool{}
	for key, v := range obj {
		target, known := mapping[key]
		if !known {
			target = key
		}
		if seen[strings.ToLower(target)] {
			return nil, bad
		}
		seen[strings.ToLower(target)] = true
		canonical := key
		if !encode {
			canonical = target
		}
		if known {
			if child, many := wireChild(shape, canonical); child != "" {
				var err error
				v, err = f.transform(child, many, v, encode)
				if err != nil {
					return nil, err
				}
			}
			if encode && shape == "search" && canonical == "name" {
				name, ok := v.(string)
				if !ok || f.Commands[name] == "" {
					return nil, bad
				}
				v = f.Commands[name]
			}
		}
		out[target] = v
	}
	return out, nil
}

// Convert is used only by the host after a validated declaration. It cannot
// introduce a field value, except a declared name for a confirmed search action.
func (f WireFormat) Convert(connector, operation string, encode bool, data []byte) ([]byte, error) {
	if err := f.Validate(connector); err != nil {
		return nil, err
	}
	if _, ok := operations(connector)[operation]; !ok {
		return nil, errors.New("unknown operation")
	}
	shape, array := wireShape(operation, encode)
	if shape == "" {
		return nil, errors.New("operation has no payload")
	}
	if len(data) > 8<<20 {
		return nil, errors.New("JSON size limit")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	budget := 200000
	value, err := readWireJSON(d, 0, &budget)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errors.New("missing acquisition payload")
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, errors.New("trailing JSON")
	}
	value, err = f.transform(shape, array, value, encode)
	if err != nil {
		return nil, err
	}
	out, err := json.Marshal(value)
	if err != nil || len(out) > 8<<20 {
		return nil, errors.New("JSON size limit")
	}
	return out, nil
}
