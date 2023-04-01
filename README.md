# prom_wrapper_2

Native pull method...

Metrics specified as a single variable typed as a struct

- Start Prometheus
docker run \
    -p 9090:9090 \
    -v /Users/george/Desktop/ProjectsCommon/prometheus/config:/etc/prometheus \
    prom/prometheus


- Start Grafana
docker run -p 3000:3000 grafana/grafana-enterprise