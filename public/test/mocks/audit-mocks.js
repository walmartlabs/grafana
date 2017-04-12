define([],
  function() {
  'use strict';

  return {
    versions: function() {
      return [{
        id: 1,
        dashboardId: 1,
        slug: 'audit-dashboard',
        version: 4,
        created: '2017-02-22T17:06:37-08:00',
        createdBy: 0,
        message: '',
      },
      {
        id: 2,
        dashboardId: 1,
        slug: 'audit-dashboard',
        version: 5,
        created: '2017-02-22T17:29:52-08:00',
        createdBy: 0,
        message: '',
      },
      {
        id: 3,
        dashboardId: 1,
        slug: 'audit-dashboard',
        version: 6,
        created: '2017-02-22T17:43:01-08:00',
        createdBy: 0,
        message: '',
      }];
    },
    compare: function() {
      return {
        delta: {
          Created: [
            '2017-02-22T17:29:52-08:00',
            '2017-02-23T22:31:33-08:00',
          ],
          CreatedBy: [0, 1],
          Data: {
            annotations: {
              list: {
                '_0': [
                  {
                    datasource: '-- Grafana --',
                    enable: true,
                    iconColor: 'rgba(255, 96, 96, 1)',
                    limit: 100,
                    name: 'foo',
                    type: 'alert',
                  },
                  0,
                  0,
                ],
                _t: 'a',
              }
            },
            editMode: [false],
            rows: {
              0: {
                panels: {
                  0: {
                    links: [
                      []
                    ],
                    targets: {
                      0: {
                        alias: [''],
                        aliasMode: ['default'],
                        downsampling: ['avg'],
                        errors: [{ 'metric': 'You must supply a metric name.' }],
                        groupBy: [{ 'timeInterval': '1s' }],
                        horAggregator: [
                          {
                            factor: '1',
                            percentile: '0.75',
                            samplingRate: '1s',
                            trim: 'both',
                            unit: 'millisecond',
                          }
                        ],
                        refId: ['A'],
                      },
                      _t: 'a'
                    },
                    title: ['Panel Title', 'CPU']
                  },
                  _t: 'a'
                }
              },
              _t: 'a'
            },
            version: [5, 7]
          },
          Id: [2, 4],
          Version: [5, 7]
        },
        meta: {
          new: 7,
          original: 5
        }
      };
    },
    restore: function(version, restoredFrom) {
      return {
        dashboard: {
          meta: {
            type: 'db',
            canSave: true,
            canEdit: true,
            canStar: true,
            slug: 'audit-dashboard',
            expires: '0001-01-01T00:00:00Z',
            created: '2017-02-21T18:40:45-08:00',
            updated: '2017-04-11T21:31:22.59219665-07:00',
            updatedBy: 'admin',
            createdBy: 'admin',
            version: version,
          },
          dashboard: {
            annotations: {
              list: []
            },
            description: 'A random dashboard for implementing the audit log',
            editable: true,
            gnetId: null,
            graphTooltip: 0,
            hideControls: false,
            id: 1,
            links: [],
            restoredFrom: restoredFrom,
            rows: [{
                collapse: false,
                height: '250px',
                panels: [{
                  aliasColors: {},
                  bars: false,
                  datasource: null,
                  fill: 1,
                  id: 1,
                  legend: {
                    avg: false,
                    current: false,
                    max: false,
                    min: false,
                    show: true,
                    total: false,
                    values: false
                  },
                  lines: true,
                  linewidth: 1,
                  nullPointMode: "null",
                  percentage: false,
                  pointradius: 5,
                  points: false,
                  renderer: 'flot',
                  seriesOverrides: [],
                  span: 12,
                  stack: false,
                  steppedLine: false,
                  targets: [{}],
                  thresholds: [],
                  timeFrom: null,
                  timeShift: null,
                  title: 'Panel Title',
                  tooltip: {
                    shared: true,
                    sort: 0,
                    value_type: 'individual'
                  },
                  type: 'graph',
                  xaxis: {
                    mode: 'time',
                    name: null,
                    show: true,
                    values: []
                  },
                  yaxes: [{
                    format: 'short',
                    label: null,
                    logBase: 1,
                    max: null,
                    min: null,
                    show: true
                  }, {
                    format: 'short',
                    label: null,
                    logBase: 1,
                    max: null,
                    min: null,
                    show: true
                  }]
                }],
                repeat: null,
                repeatIteration: null,
                repeatRowId: null,
                showTitle: false,
                title: 'Dashboard Row',
                titleSize: 'h6'
              }
            ],
            schemaVersion: 14,
            style: 'dark',
            tags: [
              'development'
            ],
            templating: {
              'list': []
            },
            time: {
              from: 'now-6h',
              to: 'now'
            },
            timepicker: {
              refresh_intervals: [
                '5s',
                '10s',
                '30s',
                '1m',
                '5m',
                '15m',
                '30m',
                '1h',
                '2h',
                '1d',
              ],
              time_options: [
                '5m',
                '15m',
                '1h',
                '6h',
                '12h',
                '24h',
                '2d',
                '7d',
                '30d'
              ]
            },
            timezone: 'utc',
            title: 'Audit Dashboard',
            version: version,
          }
        },
        message: 'Dashboard restored to version ' + version,
        version: version
      };
    },
  };
});
