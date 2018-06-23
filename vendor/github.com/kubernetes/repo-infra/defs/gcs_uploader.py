#!/usr/bin/env python

# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
import atexit
import os
import os.path
import shutil
import subprocess
import sys
import tempfile

def _workspace_status_dict(root):
    d = {}
    for f in ("stable-status.txt", "volatile-status.txt"):
        with open(os.path.join(root, f)) as info_file:
            for info_line in info_file:
                info_line = info_line.strip("\n")
                key, value = info_line.split(" ")
                d[key] = value
    return d

def main(argv):
    scratch = tempfile.mkdtemp(prefix="bazel-gcs.")
    atexit.register(lambda: shutil.rmtree(scratch))

    workspace_status = _workspace_status_dict(argv.root)
    with open(argv.manifest) as manifest:
        for artifact in manifest:
            artifact = artifact.strip("\n")
            src_file, dest_dir = artifact.split("\t")
            dest_dir = dest_dir.format(**workspace_status)
            scratch_dest_dir = os.path.join(scratch, dest_dir)
            try:
                os.makedirs(scratch_dest_dir)
            except (OSError):
                # skip directory already exists errors
                pass

            src = os.path.join(argv.root, src_file)
            dest = os.path.join(scratch_dest_dir, os.path.basename(src_file))
            os.symlink(src, dest)

    ret = 0
    uploaded_paths = []
    for gcs_path in argv.gcs_paths:
        gcs_path = gcs_path.format(**workspace_status)
        local_path = None
        if gcs_path.startswith("file://"):
            local_path = gcs_path[len("file://"):]
        elif "://" not in gcs_path:
            local_path = gcs_path
        if local_path and not os.path.exists(local_path):
            os.makedirs(local_path)

        cmd = ["gsutil"]
        # When rsyncing to a local directory, parallelization thrashes the disk.
        # It also seems to be buggy, causing frequent "File exists" errors.
        # To mitigate, only use parallel mode when rsyncing to a remote path.
        if not local_path:
            cmd.append("-m")
        cmd.extend(["rsync", "-C", "-r", scratch, gcs_path])
        ret |= subprocess.call(cmd)

        uploaded_paths.append(gcs_path)

    print "Uploaded to %s" % " ".join(uploaded_paths)
    sys.exit(ret)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Upload build targets to GCS.')

    parser.add_argument("--manifest", required=True, help="path to manifest of targets")
    parser.add_argument("--root", required=True, help="path to root of workspace")
    parser.add_argument("gcs_paths", nargs="+", help="path in gcs to push targets")

    main(parser.parse_args())
