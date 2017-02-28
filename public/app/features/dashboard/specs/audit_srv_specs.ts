import {describe, beforeEach, it, sinon, expect, angularMocks} from 'test/lib/common';

import helpers from 'test/specs/helpers';
import AuditSrv from '../audit/audit_srv';

describe('auditSrv', function() {
  var ctx = new helpers.ServiceTestContext();

  var versionsResponse = [
    {
      "id": 1,
      "dashboardId": 1,
      "slug": "audit-dashboard",
      "version": 4,
      "created": "2017-02-22T17:06:37-08:00",
      "createdBy": 0,
      "message": ""
    },
    {
      "id": 2,
      "dashboardId": 1,
      "slug": "audit-dashboard",
      "version": 5,
      "created": "2017-02-22T17:29:52-08:00",
      "createdBy": 0,
      "message": ""
    },
    {
      "id": 3,
      "dashboardId": 1,
      "slug": "audit-dashboard",
      "version": 6,
      "created": "2017-02-22T17:43:01-08:00",
      "createdBy": 0,
      "message": ""
    },
  ];

  var compareResponse = {
    "delta": {
      "Created": [
        "2017-02-22T17:29:52-08:00",
        "2017-02-23T22:31:33-08:00"
      ],
      "CreatedBy": [
        0,
        1
      ],
      "Data": {
        "annotations": {
          "list": {
            "_0": [
              {
                "datasource": "-- Grafana --",
                "enable": true,
                "iconColor": "rgba(255, 96, 96, 1)",
                "limit": 100,
                "name": "foo",
                "type": "alert"
              },
              0,
              0
            ],
            "_t": "a"
          }
        },
        "editMode": [
          false
        ],
        "rows": {
          "0": {
            "panels": {
              "0": {
                "links": [
                  []
                ],
                "targets": {
                  "0": {
                    "alias": [
                      ""
                    ],
                    "aliasMode": [
                      "default"
                    ],
                    "downsampling": [
                      "avg"
                    ],
                    "errors": [
                      {
                        "metric": "You must supply a metric name."
                      }
                    ],
                    "groupBy": [
                      {
                        "timeInterval": "1s"
                      }
                    ],
                    "horAggregator": [
                      {
                        "factor": "1",
                        "percentile": "0.75",
                        "samplingRate": "1s",
                        "trim": "both",
                        "unit": "millisecond"
                      }
                    ],
                    "refId": [
                      "A"
                    ]
                  },
                  "_t": "a"
                },
                "title": [
                  "Panel Title",
                  "CPU"
                ]
              },
              "_t": "a"
            }
          },
          "_t": "a"
        },
        "version": [
          5,
          7
        ]
      },
      "Id": [
        2,
        4
      ],
      "Version": [
        5,
        7
      ]
    },
    "meta": {
      "new": 7,
      "original": 5
    }
  };

  var restoreResponse = function(version: number) {
    return {
      version,
      message: "Dashboard restored!",
    };
  };

  beforeEach(angularMocks.module('grafana.core'));
  beforeEach(angularMocks.module('grafana.services'));
  beforeEach(angularMocks.inject(function($httpBackend) {
    ctx.$httpBackend = $httpBackend;
    $httpBackend.whenRoute('GET', 'api/dashboards/db/:id/versions').respond(versionsResponse);
    $httpBackend.whenRoute('GET', 'api/dashboards/db/:id/compare/:original...:new').respond(compareResponse);
    $httpBackend.whenRoute('POST', 'api/dashboards/db/:id/restore')
      .respond(function(method, url, data, headers, params) {
        const parsedData = JSON.parse(data);
        return [200, restoreResponse(parsedData.version)];
      });
  }));
  beforeEach(ctx.createService('auditSrv'));

  describe('getAuditLog', function() {
    it('should return a versions array for the given dashboard id', function(done) {
      ctx.service.getAuditLog({ id: 1 }).then(function(versions) {
        expect(versions).to.eql(versionsResponse);
        done();
      });
      ctx.$httpBackend.flush();
    });

    it('should return an empty array when not given an id', function(done) {
      ctx.service.getAuditLog({ }).then(function(versions) {
        expect(versions).to.eql([]);
        done();
      });
      ctx.$httpBackend.flush();
    });

    it('should return an empty array when not given a dashboard', function(done) {
      ctx.service.getAuditLog().then(function(versions) {
        expect(versions).to.eql([]);
        done();
      });
      ctx.$httpBackend.flush();
    });
  });

  describe('compareVersions', function() {
    it('should return a diff object for the given dashboard revisions', function(done) {
      var compare = { original: 5, new: 7 };
      ctx.service.compareVersions({ id: 1 }, compare).then(function(response) {
        expect(response).to.eql(compareResponse);
        done();
      });
      ctx.$httpBackend.flush();
    });

    it('should return an empty object when not given an id', function(done) {
      var compare = { original: 5, new: 7 };
      ctx.service.compareVersions({ }, compare).then(function(response) {
        expect(response).to.eql({});
        done();
      });
      ctx.$httpBackend.flush();
    });
  });

  describe('restoreDashboard', function() {
    it('should return a success response given valid parameters', function(done) {
      var version = 6;
      ctx.service.restoreDashboard({ id: 1 }, version).then(function(response) {
        expect(response).to.eql(restoreResponse(version));
        done();
      });
      ctx.$httpBackend.flush();
    });

    it('should return an empty object when not given an id', function(done) {
      ctx.service.restoreDashboard({}, 6).then(function(response) {
        expect(response).to.eql({});
        done();
      });
      ctx.$httpBackend.flush();
    });
  });
});
