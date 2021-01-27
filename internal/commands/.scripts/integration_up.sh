echo "Setting auth keys"
export BASE_URI="http://ast.cxflow-ci.com"
export AST_ACCESS_KEY_ID="ast-plugins-3d1b8499-b3cf-43cf-8425-0bb212ca44d3"
export AST_ACCESS_KEY_SECRET="c710a92f-d103-4bc5-89ce-6d138f976b9e"
echo $AST_ACCESS_KEY_ID
echo $BASE_URI
echo $AST_ACCESS_KEY_SECRET
echo "Done setting keys, trigginer process"
go test ./test/integration/scan_test.go \
            ./test/integration/root_test.go \
            ./test/integration/project_test.go \
            ./test/integration/health_check_test.go \
            ./test/integration/result_test.go \
            ./test/integration/sast_resources_test.go -v