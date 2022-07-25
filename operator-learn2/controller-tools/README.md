##生成types
type-scaffold --kind Foo

##根据types生成deepcopy
controller-gen object paths=./pkg/apis/zqa.test/v1/types.go