This directory contains scripts to run a quick demo of Kubebuilder.

Steps to run demo:

```sh
mkdir /tmp/kb-demo
cd /tmp/kb-demo
DEMO_AUTO_RUN=1 ./run.sh

```

Instructions for producing the demo movie:

```sh

# Create temporary directory
mkdir /tmp/kb-demo
cd /tmp/kb-demo

asciinema rec
<path-to-KB-repo>/scripts/demo/run.sh

<CTRL-C> to terminate the script
<CTRL-D> to terminate the asciinema recording
<CTRL-C> to save the recording locally

# Edit the recorded file by editing the controller-gen path
# Once you are happy with the recording, use svg-term program to generate the svg

svg-term --cast=<movie-id> --out demo.svg --window
```
