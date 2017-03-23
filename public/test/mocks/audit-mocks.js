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
    restore: function(version) {
      return {
        version: version,
        message: "Dashboard restored!",
      };
    },
  };
});
