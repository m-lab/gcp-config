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

# NOTE: This will take a very long time, first "rsync" == "cp".
# NOTE: The old directories should be removed after the rsync completes.

# NDT web100
for year in 2009 2010 2011 2012 2013 2014 2015 2016 2017 2018 2019 ; do
  gsutil -m rsync -r gs://${archive}/ndt/${year}/ gs://${archive}/ndt/web100/${year}/
done

# Sidestream web100
for year in 2009 2010 2011 2012 2013 2014 2015 2016 2017 2018 2019 ; do
  gsutil -m rsync -r gs://${archive}/sidestream/${year}/ gs://${archive}/vserver/sidestream/${year}/
done

# Paris-traceroute
for year in 2013 2014 2015 2016 2017 2018 2019 ; do
  gsutil -m rsync -r gs://${archive}/paris-traceroute/${year}/ gs://${archive}/vserver/traceroute/${year}/
done

# Switch
for year in 2016 2017 2018 2019 ; do
  gsutil -m rsync -r gs://${archive}/switch/${year}/ gs://${archive}/utilization/switch/${year}/
done

# TODO: wait until ndt-server creates the single ndt7 directory name.
# i.e. https://github.com/m-lab/ndt-server/pull/264 is in production.
for year in 2019 2020 ; do
  # NDT7 Upload / Download
  gsutil -m rsync -r gs://${archive}/ndt/ndt7/upload/${year}/ gs://${archive}/ndt/ndt7/${year}/
  gsutil -m rsync -r gs://${archive}/ndt/ndt7/download/${year}/ gs://${archive}/ndt/ndt7/${year}/
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
  gsutil -m rm -r gs://${archive}/ndt/${year}/
done

# Sidestream web100
for year in 2009 2010 2011 2012 2013 2014 2015 2016 2017 2018 2019 ; do
  gsutil -m rm -r gs://${archive}/sidestream/${year}/
done

# Paris-traceroute
for year in 2013 2014 2015 2016 2017 2018 2019 ; do
  gsutil -m rm -r gs://${archive}/paris-traceroute/${year}/
done

# Switch
for year in 2016 2017 2018 2019 ; do
  gsutil -m rm -r gs://${archive}/switch/${year}/
done

# TODO: wait until ndt-server creates the single ndt7 directory name.
# i.e. https://github.com/m-lab/ndt-server/pull/264 is in production.
for year in 2019 2020 ; do
  # NDT7 Upload / Download
  gsutil -m rm -r gs://${archive}/ndt/ndt7/upload/${year}/
  gsutil -m rm -r gs://${archive}/ndt/ndt7/download/${year}/
done
