#!/bin/bash -x
set -e

ARTIFACT_DIR=${ARTIFACT_DIR:-_output}
manifest=${ARTIFACT_DIR}/manifest.yaml
global_manifest=${ARTIFACT_DIR}/global_manifest.yaml
rm -f $manifest $global_manifest
mkdir -p ${ARTIFACT_DIR}

if [ -n "${IMAGE_FORMAT}" ] ; then
    echo "IMAGE_FORMAT set as '${IMAGE_FORMAT}'"
    IMAGE_LOCAL_STORAGE_OPERATOR=$(sed -e "s,\${component},local-storage-operator," <(echo $IMAGE_FORMAT))
    IMAGE_LOCAL_DISKMAKER=$(sed -e "s,\${component},local-storage-diskmaker," <(echo $IMAGE_FORMAT))
else
    IMAGE_LOCAL_STORAGE_OPERATOR=${IMAGE_LOCAL_STORAGE_OPERATOR:-quay.io/openshift/origin-local-storage-operator}
    IMAGE_LOCAL_DISKMAKER=${IMAGE_LOCAL_DISKMAKER:-quay.io/openshift/origin-local-storage-diskmaker}
fi

KUBECONFIG=${KUBECONFIG:-$HOME/.kube/config}
repo_dir="$(dirname $0)/.."
cat ${repo_dir}/deploy/sa.yaml >> ${manifest}
cat ${repo_dir}/deploy/rbac.yaml >> ${manifest}
cat ${repo_dir}/deploy/operator.yaml >> ${manifest}
cat ${repo_dir}/deploy/localvolume_crd.yaml >> ${global_manifest}
cat ${repo_dir}/deploy/localvolumeset_crd.yaml >> ${global_manifest}
cat ${repo_dir}/deploy/localvolumediscovery_crd.yaml >> ${global_manifest}
cat ${repo_dir}/deploy/localvolumediscoveryresult_crd.yaml >> ${global_manifest}

sed -i "s,quay.io/openshift/origin-local-storage-operator,${IMAGE_LOCAL_STORAGE_OPERATOR}," ${manifest}
sed -i "s,quay.io/openshift/origin-local-storage-diskmaker,${IMAGE_LOCAL_DISKMAKER}," ${manifest}
NAMESPACE=${NAMESPACE:-default}
LOCAL_DISK=${LOCAL_DISK:-""}

export \
    IMAGE_LOCAL_STORAGE_OPERATOR \
    IMAGE_LOCAL_DISKMAKER

TEST_NAMESPACE=${NAMESPACE} TEST_LOCAL_DISK=${LOCAL_DISK} go test -timeout 0 ./test/e2e/... \
  -root=$(pwd) \
  -kubeconfig=${KUBECONFIG} \
  -globalMan ${global_manifest} \
  -namespacedMan ${manifest} \
  -v \
  -parallel=1 \
  -singleNamespace
