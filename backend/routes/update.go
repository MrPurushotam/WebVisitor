package routes

// import (
// 	"os"
// 	"github.com/gin-gonic/gin"
// )

// var binId = os.Getenv("JSON_BIN_ID")

// func fetchJsonData(binId string)(string,error){
// 	url := 	"https://json.extendsclass.com/bin/" + binId
// 	req,err:= http.NewRequest("GET",url,null)
// 	if err != nil{
// 		return "",err
// 	}

// 	client := $http.client{}
// 	res,err:=client.Do(req);
// 	if(err!=nil){
// 		return "",err
// 	}
// 	defer resp.Body.close()
// 	body,err := ioutil.ReadAll(resp.Body)
// 	if err != nil{
// 		return "",err
// 	}
// 	return string(body),nil
// }

// func getUrls(c *gin.Context) {
	
// }

// func deleteUrl(c *gin.Context) {

// }

// func updateUrl(c *gin.Context) {

// }

// func RoutesInit(router *gin.Engine) {
// 	router.GET("/url/", getUrls)
// 	router.DELETE("/url/", deleteUrl)
// 	router.PUT("/url/", updateUrl)
// }
