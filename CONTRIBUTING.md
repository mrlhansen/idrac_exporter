# How to contribute
Contributions to the project are welcome, but please follow the guidelines below.

* Open an issue for all bug reports. Whenever possible, please include the following information.

  * The log output from the exporter.

  * The machine type (vendor and model) that you are trying to export metrics from.

  * Relevant data from the Redfish API. When using the `-debug` argument, the exporter will dump all responses from the Redfish API in JSON format.


* Code contributions are accepted. However, please open an issue to discuss the changes before submitting the pull request.

  * Pull requests without an associated issue might simply be closed without explanation.

  * The exporter aims to support metrics that are widely available across vendors. Smaller hacks to improve support for a specific vendor are fine, but implementing bigger changes, that only works for a single vendor, are likely not going to be accepted.

  * Keep in mind that all functionality added to the exporter has to be maintained by the repository owner once merged. This is why changes should be discussed beforehand.
