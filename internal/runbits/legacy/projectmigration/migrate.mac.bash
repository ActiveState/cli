sed -E -i '' 's/(\?|&)commitID=[a-zA-Z0-9-]*//' "$project.path()/activestate.yaml"
echo "Project successfully migrated. You can now delete the migrate-to-buildscripts script from your activestate.yaml"
