package rest

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
	"strings"
)

type QueryParams struct {
	IsReference bool `json:"is_reference"`
	// Sort 排序 字段
	Sort string `json:"sort"`
	// Order 排序方式 DESC/
	Order string `json:"order"`
	// OrderType 排序方式 -1/1
	OrderType int `json:"order_type"`
	// OrderType 排序方式 -1/1
	Page int64 `json:"page"`
	// OrderType 排序方式 -1/1
	PerPage int64 `json:"per_page"`
	// OrderType 排序方式 -1/1
	Filter map[string]interface{} `json:"filter"`
}

// LoadQuery
// 页面参数
// ?filter={}&order=DESC&page=1&perPage=10&sort=created_at
// ?filter={"verified":false}&order=DESC&page=1&perPage=10&sort=created_at
// ?filter={"verified":false,"truck_category_id":"6515a2ac9f46f389d6804c86"}&order=DESC&page=1&perPage=10&sort=created_at
// 接口参数
// ?filter={"verified":false,"truck_category_id":"6515a2ac9f46f389d6804c86"}&range=[0,9]&sort=["id","ASC"]
func LoadQuery(c *gin.Context) QueryParams {

	q := QueryParams{}

	// 加载分页
	rangeString := c.DefaultQuery("range", "[0, 9]")
	var rangeValue []int
	err := json.Unmarshal([]byte(rangeString), &rangeValue)
	if err != nil {
		log.WithField("range", rangeString).Error(err)
		q.PerPage = 10
		q.Page = 1
	}

	// 页码转换
	if len(rangeValue) == 2 {
		limit := rangeValue[1] - rangeValue[0] // 19 - 10
		total := rangeValue[1]
		q.PerPage = int64(limit)
		q.Page = int64(total)%int64(limit) + 1 // 1
	}

	// 加载排序
	sortString := c.DefaultQuery("sort", `["id","ASC"]`)
	var sortValue []string
	err = json.Unmarshal([]byte(sortString), &sortValue)
	if err != nil {
		log.WithField("sort", sortValue).Error(err)
		q.Order = "ASC"
		q.Sort = "id"
	}

	if len(sortValue) == 2 {
		q.Order = sortValue[1]
		q.Sort = sortValue[0]
	}

	if q.Order == "ASC" {
		q.OrderType = -1
	} else {
		q.OrderType = 1
	}

	// 加载过滤器
	filterString := c.DefaultQuery("filter", "{}")
	var filterValue map[string]interface{}
	err = json.Unmarshal([]byte(filterString), &filterValue)
	if err != nil {
		log.WithField("filter", filterString).Error(err)
	}
	q.Filter = filterValue

	// 转换为mongo过滤器
	return q
}

// AsMongoFilter 转换过滤器
// fields: a=b, b=c
func (q QueryParams) AsMongoFilter(fields []string, filters map[string]interface{}) (interface{}, options.FindOptions) {
	mongoFilters := bson.D{}
	inFilters := bson.M{}
	for _, key := range fields {
		// 字段转换 例如 ?filter={"id":["6502adb4529dbe1ee8f07457","6502ad86529dbe1ee8f07441"]
		// fields 通过id 获取到列表 ["6502adb4529dbe1ee8f07457","6502ad86529dbe1ee8f07441"]
		// 通过转换key， 例如mongodb 内部使用到的是 _id 而不是 id
		// 所以使用 finalKey
		// 如果没有转换，则使用原始的
		keyWithRename := strings.Split(key, "=>") // a=>b
		originKey := keyWithRename[0]
		finalKey := originKey
		if len(keyWithRename) == 2 {
			// 需要转换
			finalKey = keyWithRename[1]
		}
		val, ok := filters[originKey]
		if ok {
			if InterfaceIsSlice(val) {
				objIds := q.toObjectID(val)
				if len(objIds) > 0 {
					q.IsReference = true
					inFilters := bson.M{}
					inFilters[finalKey] = bson.M{"$in": objIds}
				}
			} else {
				filter := bson.E{Key: finalKey, Value: val}
				mongoFilters = append(mongoFilters, filter)
			}
		}

	}

	// 设置查询选项
	findOptions := options.FindOptions{}
	findOptions.SetSort(bson.D{{q.Sort, q.OrderType}})
	findOptions.SetSkip((q.Page - 1) * q.PerPage)
	findOptions.SetLimit(q.PerPage)

	if q.IsReference {
		return inFilters, options.FindOptions{}
	}
	return mongoFilters, findOptions
}

func (q QueryParams) toObjectID(v interface{}) []*primitive.ObjectID {

	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	var slice []string
	err = json.Unmarshal(b, &slice)
	if err != nil {
		panic(err)
	}
	// 统一转换所有引用的id
	objIds := make([]*primitive.ObjectID, 0)
	for _, i := range slice {
		objID, _ := primitive.ObjectIDFromHex(i)
		objIds = append(objIds, &objID)
	}
	return objIds
}

// RangeContent 分页显示 1, 2, 3 ...99
func (q QueryParams) RangeContent(counter int64) string {
	// 返回数据列表包含分页头部信息
	a := (q.Page - 1) * q.PerPage
	b := (q.Page-1)*q.PerPage + q.PerPage
	return fmt.Sprintf("items %d-%d/%d", a, b, counter)
}

// Reference 引用
// ?filter={"id":["6502adb4529dbe1ee8f07457", "6502aab7529dbe1ee8f072a7"]}
func (q QueryParams) Reference() {

}

func InterfaceIsSlice(t interface{}) bool {
	switch reflect.TypeOf(t).Kind() {
	case reflect.Slice:
		return true
	default:
		return false
	}
}
