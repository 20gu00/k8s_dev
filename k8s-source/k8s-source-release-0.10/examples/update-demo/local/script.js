/*
Copyright 2014 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

var base = "http://localhost:8001/api/v1beta1/";

var updateImage = function($http, server) {
  $http.get("http://" + server.ip + ":8080/data.json")
    .success(function(data) {
      server.image = data.image;
      console.log(data);
    })
    .error(function(data) {
      server.image = "";
      console.log(data);
    });
};

var updateServer = function($http, server) {
  $http.get(base + "pods/" + server.id)
    .success(function(data) {
      console.log(data);
      server.ip = data.currentState.hostIP;
      server.labels = data.labels;
      server.host = data.currentState.host.split('.')[0];
      server.status = data.currentState.status;

      server.dockerImage = data.currentState.info["update-demo"].Image;
      updateImage($http, server);
    })
    .error(function(data) {
      console.log(data);
    });
};

var updateData = function($scope, $http) {
  var servers = $scope.servers;
  for (var i = 0; i < servers.length; ++i) {
    var server = servers[i];
    updateServer($http, server);
  }
};

var ButtonsCtrl = function ($scope, $http, $interval) {
  $scope.servers = [];
  update($scope, $http);
  $interval(angular.bind({}, update, $scope, $http), 2000);
};

var getServer = function($scope, id) {
  var servers = $scope.servers;
  for (var i = 0; i < servers.length; ++i) {
    if (servers[i].id == id) {
      return servers[i];
    }
  }
  return null;
};

var isUpdateDemoPod = function(pod) {
    return pod.labels && pod.labels.name == "update-demo";
};

var update = function($scope, $http) {
  if (!$http) {
    console.log("No HTTP!");
    return;
  }
  $http.get(base + "pods")
    .success(function(data) {
      console.log(data);
      var newServers = [];
      for (var i = 0; i < data.items.length; ++i) {
        var pod = data.items[i];
        if (!isUpdateDemoPod(pod)) {
          continue;
        }
        var server = getServer($scope, pod.id);
        if (server == null) {
          server = { "id": pod.id };
        }
        newServers.push(server);
      }
      $scope.servers = newServers;
      updateData($scope, $http);
    })
    .error(function(data) {
      console.log("ERROR: " + data);
    })
};
