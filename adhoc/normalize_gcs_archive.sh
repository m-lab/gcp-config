#!/bin/bash
#
# normalize_gcs_archive.sh is a single-use script that will sync legacy
# datatypes from the <exp>/YYYY pattern to <exp>/<datatype>/YYYY pattern.
#
# New data is not being added to the legacy locations. So, running rsync once
# should be sufficient. However, the steps below run it twice to guarantee that
# all data is in place.
#
# Steps:
# * rsync data from old to new locations (slow)
# * delete exits below.
# * rsync again to confirm all data is copied (faster)
# * remove data from old locations (slower)


set -e
set -x

archive=${1:?Please provide GCS bucket name, e.g. archive-mlab-sandbox}

function assert_src_files_found_in_dst() {
  local src=$1
  local dst=$2

  # We cannot compare the files directly b/c the object paths have changed.
  # Extract just the archive name from the last path component.
  awk -F/ '{print $NF}' $src | sort > src.files
  awk -F/ '{print $NF}' $dst | sort > dst.files

  # Report files that are unique to the src. There should be zero.
  c=`comm -2 -3 src.files dst.files | wc -l`
  if [[ $c != "0" ]]; then
    echo "FAILURE:"
    comm -2 -3 src.files dst.files
    return 1
  else
    echo "SUCCESS: all files from $src found in $dst"
    return 0
  fi
}

function safe_rsync() {
  local src=$1
  local dst=$2

  # List all files in the src, to compare to all files in dst at the end.
  time gsutil ls -r $src > src.raw
  # Perform copy.
  time gsutil -m rsync -r $src $dst
  # List all files in the dst (which should include src files now).
  time gsutil ls -r $dst > dst.raw
  # Assert that the src files are found in the dst files.
  assert_src_files_found_in_dst src.raw dst.raw
}

# NOTE: This will take a very long time. Each "safe_rsync" performs several steps:
# * list files in src
# * run rsync
# * list files in dst
# * verify that src files are found in dst.

# NDT web100
for year in 2009 2010 2011 2012 2013 2014 2015 2016 2017 2018 2019 ; do
  safe_rsync gs://${archive}/ndt/${year}/ gs://${archive}/ndt/web100/${year}/
done

# Sidestream web100
for year in 2009 2010 2011 2012 2013 2014 2015 2016 2017 2018 2019 ; do
  safe_rsync gs://${archive}/sidestream/${year}/ gs://${archive}/host/sidestream/${year}/
done

# Switch
for year in 2016 2017 2018 2019 ; do
  safe_rsync gs://${archive}/switch/${year}/ gs://${archive}/utilization/switch/${year}/
done

# TODO: wait until ndt-server creates the single ndt7 directory name.
# i.e. https://github.com/m-lab/ndt-server/pull/264 is in production.
for year in 2019 2020 ; do
  # NDT7 Upload / Download
  safe_rsync gs://${archive}/ndt/ndt7/upload/${year}/ gs://${archive}/ndt/ndt7/${year}/
  safe_rsync gs://${archive}/ndt/ndt7/download/${year}/ gs://${archive}/ndt/ndt7/${year}/
done

# ndt/host/neubot traceroute.
for year in 2019 2020 2021 ; do
  safe_rsync gs://${archive}/ndt/traceroute/${year}/ gs://${archive}/ndt/scamper1/${year}/
  safe_rsync gs://${archive}/host/traceroute/${year}/ gs://${archive}/host/scamper1/${year}/
  safe_rsync gs://${archive}/neubot/traceroute/${year}/ gs://${archive}/neubot/scamper1/${year}
done

# Delete exits to proceed to remove legacy folders.
exit 0
exit 0
exit 0
exit 0


################################################################################
# REMOVE
################################################################################

# NDT web100
for year in 2009 2010 2011 2012 2013 2014 2015 2016 2017 2018 2019 ; do
  time gsutil -m rm -r gs://${archive}/ndt/${year}/
done

# Sidestream web100
for year in 2009 2010 2011 2012 2013 2014 2015 2016 2017 2018 2019 ; do
  time gsutil -m rm -r gs://${archive}/sidestream/${year}/
done

# Switch
for year in 2016 2017 2018 2019 ; do
  time gsutil -m rm -r gs://${archive}/switch/${year}/
done

# TODO: wait until ndt-server creates the single ndt7 directory name.
# i.e. https://github.com/m-lab/ndt-server/pull/264 is in production.
for year in 2019 2020 ; do
  # NDT7 Upload / Download
  time gsutil -m rm -r gs://${archive}/ndt/ndt7/upload/${year}/
  time gsutil -m rm -r gs://${archive}/ndt/ndt7/download/${year}/
done
