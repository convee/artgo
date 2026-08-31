// artgo basic example static asset.
(function () {
  var button = document.querySelector("#check");
  var result = document.querySelector("#result");

  button.addEventListener("click", function () {
    result.textContent = "loading...";
    fetch("/api/health")
      .then(function (response) {
        if (!response.ok) {
          throw new Error("HTTP " + response.status);
        }
        return response.json();
      })
      .then(function (data) {
        result.textContent = JSON.stringify(data, null, 2);
      })
      .catch(function (error) {
        result.textContent = "request failed: " + error.message;
      });
  });
})();
