/*
 * 查询接口
 *
 * wencan
 * 2018-12-26
 */

package cachex

// QueryFunc 查询过程签名
type QueryFunc func(request, value interface{}) error

// Query 查询过程实现Querier接口
func (fun QueryFunc) Query(request, value interface{}) error {
	return fun(request, value)
}

// Querier 查询接口
type Querier interface {
	// Query 查询。value必须是非nil指针。没找到返回NotFound错误实现
	Query(request, value interface{}) error
}
