package api

import (
	"time"

	"github.com/grafana/grafana/pkg/components/simplejson"
	m "github.com/grafana/grafana/pkg/models"

	jsondiff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
)

// diff compares two simplejson values as bytes, and returns the diff.
func diff(originalJSON *simplejson.Json, newJSON *simplejson.Json) (*simplejson.Json, error) {
	x, err := originalJSON.Encode()
	if err != nil {
		return nil, err
	}
	y, err := newJSON.Encode()
	if err != nil {
		return nil, err
	}

	// From the raw bytes, run the JSON diff from the package
	differ := jsondiff.New()
	diff, err := differ.Compare(x, y)
	if err != nil {
		return nil, err
	}

	// If no change has been made, exit
	// TODO(ben) should error with appropriate message
	if !diff.Modified() {
		return nil, nil
	}

	// Otherwise returns the change
	format := formatter.NewDeltaFormatter()
	result, err := format.FormatAsJson(diff)
	return simplejson.NewFromAny(result), err
}

// getMockData gets the mock data for the given slug and version.
func getMockData(slug string, version int) *simplejson.Json {
	data := getData(slug)
	v := data[version]
	v.Data = getJSON(v.Version)
	return simplejson.NewFromAny(data[version])
}

// getAllMockData gets all the mock data.
func getAllMockData(slug string) *simplejson.Json {
	data := getData(slug)

	// Return the data without the big data string
	return simplejson.NewFromAny(toDTO(data))
}

// toDTO transforms the mock data into the DTO format
func toDTO(src map[int]*m.DashboardVersion) map[int]*m.DashboardVersionDTO {
	data := make(map[int]*m.DashboardVersionDTO)
	for i, d := range src {
		dto := &m.DashboardVersionDTO{}

		d.Id = dto.Id
		d.Slug = dto.Slug
		d.Version = dto.Version
		d.Created = dto.Created
		d.CreatedBy = dto.CreatedBy
		d.Message = dto.Message

		data[i] = dto
	}
	return data
}

func getData(slug string) map[int]*m.DashboardVersion {
	data := make(map[int]*m.DashboardVersion)

	data[0] = &m.DashboardVersion{
		Id:        1,
		Slug:      slug,
		Version:   0,
		Created:   time.Now().Add(time.Hour * -1),
		CreatedBy: 1,
		Message:   "Created dashboard " + slug,
	}

	data[1] = &m.DashboardVersion{
		Id:        1,
		Slug:      slug,
		Version:   1,
		Created:   time.Now().Add(time.Minute * -12),
		CreatedBy: 1,
		Message:   "Changed graph title",
	}

	data[2] = &m.DashboardVersion{
		Id:        1,
		Slug:      slug,
		Version:   2,
		Created:   time.Now().Add(time.Minute * -3),
		CreatedBy: 1,
		Message:   "Updated value for x axis",
	}

	return data
}

// getJSON returns some test JSON for mock diffs
func getJSON(version int) *simplejson.Json {
	// If we don't have the version just bail out with
	// an error message
	if version > 2 || version < 0 {
		return simplejson.NewFromAny(map[string]string{
			"error": "version not found",
		})
	}

	// Create a map for the mock JSON data
	data := make(map[int]*simplejson.Json)

	// Mock data for entry zero
	x, err := simplejson.NewJson([]byte(`{
  "annotations": {
    "list": [

    ]
  },
  "editable": true,
  "gnetId": null,
  "graphTooltip": 0,
  "hideControls": false,
  "id": 1,
  "links": [

  ],
  "rows": [
    {
      "collapse": false,
      "height": "250px",
      "panels": [
        {
          "aliasColors": {

          },
          "bars": false,
          "datasource": null,
          "description": "TEST_PANEL_DESCRIPTION",
          "fill": 1,
          "id": 1,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [

          ],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [

          ],
          "span": 12,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "refId": "A"
            }
          ],
          "thresholds": [

          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "TEST_PANEL_UPDATE",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "mode": "time",
            "name": null,
            "show": true,
            "values": [

            ]
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": false,
      "title": "Dashboard Row",
      "titleSize": "h6"
    }
  ],
  "schemaVersion": 14,
  "style": "dark",
  "tags": [

  ],
  "templating": {
    "list": [

    ]
  },
  "time": {
    "from": "now-6h",
    "to": "now"
  },
  "timepicker": {
    "refresh_intervals": [
      "5s",
      "10s",
      "30s",
      "1m",
      "5m",
      "15m",
      "30m",
      "1h",
      "2h",
      "1d"
    ],
    "time_options": [
      "5m",
      "15m",
      "1h",
      "6h",
      "12h",
      "24h",
      "2d",
      "7d",
      "30d"
    ]
  },
  "timezone": "browser",
  "title": "TEST_DASHBOARD",
  "version": 1
}
`))
	if err != nil {
		panic(err)
	}

	// Mock data for entry one - only the description has changed
	y, err := simplejson.NewJson([]byte(`{
  "annotations": {
    "list": [

    ]
  },
  "editable": true,
  "gnetId": null,
  "graphTooltip": 0,
  "hideControls": false,
  "id": 1,
  "links": [

  ],
  "rows": [
    {
      "collapse": false,
      "height": "250px",
      "panels": [
        {
          "aliasColors": {

          },
          "bars": false,
          "datasource": null,
          "description": "TEST_PANEL_DESCRIPTION_CHANGED",
          "fill": 1,
          "id": 1,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [

          ],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [

          ],
          "span": 12,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "refId": "A"
            }
          ],
          "thresholds": [

          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "TEST_PANEL_UPDATE",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "mode": "time",
            "name": null,
            "show": true,
            "values": [

            ]
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": false,
      "title": "Dashboard Row",
      "titleSize": "h6"
    }
  ],
  "schemaVersion": 14,
  "style": "dark",
  "tags": [

  ],
  "templating": {
    "list": [

    ]
  },
  "time": {
    "from": "now-6h",
    "to": "now"
  },
  "timepicker": {
    "refresh_intervals": [
      "5s",
      "10s",
      "30s",
      "1m",
      "5m",
      "15m",
      "30m",
      "1h",
      "2h",
      "1d"
    ],
    "time_options": [
      "5m",
      "15m",
      "1h",
      "6h",
      "12h",
      "24h",
      "2d",
      "7d",
      "30d"
    ]
  },
  "timezone": "browser",
  "title": "TEST_DASHBOARD",
  "version": 1
}
`))
	if err != nil {
		panic(err)
	}

	z, err := simplejson.NewJson([]byte(`{
  "annotations": {
    "list": [

    ]
  },
  "editMode": false,
  "editable": true,
  "gnetId": null,
  "graphTooltip": 0,
  "hideControls": false,
  "id": null,
  "links": [

  ],
  "rows": [
    {
      "collapse": false,
      "height": "250px",
      "panels": [
        {
          "cacheTimeout": null,
          "colorBackground": true,
          "colorValue": false,
          "colors": [
            "rgba(245, 54, 54, 0.9)",
            "rgba(237, 129, 40, 0.89)",
            "rgba(50, 172, 45, 0.97)"
          ],
          "datasource": null,
          "description": "COOL",
          "format": "none",
          "gauge": {
            "maxValue": 100,
            "minValue": 0,
            "show": false,
            "thresholdLabels": false,
            "thresholdMarkers": true
          },
          "id": 1,
          "interval": null,
          "links": [

          ],
          "mappingType": 1,
          "mappingTypes": [
            {
              "name": "value to text",
              "value": 1
            },
            {
              "name": "range to text",
              "value": 2
            }
          ],
          "maxDataPoints": 100,
          "nullPointMode": "connected",
          "nullText": null,
          "postfix": "",
          "postfixFontSize": "50%",
          "prefix": "",
          "prefixFontSize": "50%",
          "rangeMaps": [
            {
              "from": "null",
              "text": "N\/A",
              "to": "null"
            }
          ],
          "span": 12,
          "sparkline": {
            "fillColor": "rgba(31, 118, 189, 0.18)",
            "full": false,
            "lineColor": "rgb(31, 120, 193)",
            "show": false
          },
          "targets": [
            {
              "refId": "A"
            }
          ],
          "thresholds": "",
          "title": "DIFFERENT_SOURCE",
          "type": "singlestat",
          "valueFontSize": "80%",
          "valueMaps": [
            {
              "op": "=",
              "text": "value",
              "value": "key"
            }
          ],
          "valueName": "avg"
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": false,
      "title": "Dashboard Row",
      "titleSize": "h6"
    }
  ],
  "schemaVersion": 14,
  "style": "dark",
  "tags": [

  ],
  "templating": {
    "list": [

    ]
  },
  "time": {
    "from": "now-6h",
    "to": "now"
  },
  "timepicker": {
    "refresh_intervals": [
      "5s",
      "10s",
      "30s",
      "1m",
      "5m",
      "15m",
      "30m",
      "1h",
      "2h",
      "1d"
    ],
    "time_options": [
      "5m",
      "15m",
      "1h",
      "6h",
      "12h",
      "24h",
      "2d",
      "7d",
      "30d"
    ]
  },
  "timezone": "browser",
  "title": "TEST_DASHBOARD",
  "version": 2
}
`))
	if err != nil {
		panic(err)
	}

	// Mock data for entry two - everything has changed!

	data[0] = x
	data[1] = y
	data[2] = z

	return data[version]
}
