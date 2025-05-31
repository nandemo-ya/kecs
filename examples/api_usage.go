package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/nandemo-ya/kecs/internal/controlplane/api/generated"
)

func main() {
	// 生成されたECSサービスのインスタンスを作成
	ecsService := generated.NewECSService()

	// CreateCluster APIを使用する例
	ctx := context.Background()
	
	// リクエストを作成
	clusterName := "my-test-cluster"
	createClusterReq := &generated.CreateClusterRequest{
		// clusterName フィールドは生成されたスタブの型定義に従って設定
		// 注意: 実際の型定義は generated/types.go を確認してください
	}

	// API呼び出し
	fmt.Println("Calling CreateCluster API...")
	response, err := ecsService.CreateCluster(ctx, createClusterReq)
	if err != nil {
		log.Fatalf("CreateCluster failed: %v", err)
	}

	fmt.Printf("CreateCluster response: %+v\n", response)

	// HTTPハンドラーの使用例
	fmt.Println("\nSetting up HTTP handlers...")
	
	// 各APIのHTTPハンドラーを設定
	http.HandleFunc("/v1/createcluster", generated.HandleCreateCluster(ecsService))
	http.HandleFunc("/v1/listclusters", generated.HandleListClusters(ecsService))
	http.HandleFunc("/v1/describeclusters", generated.HandleDescribeClusters(ecsService))
	http.HandleFunc("/v1/runtask", generated.HandleRunTask(ecsService))
	
	// サーバー起動例
	fmt.Println("Starting server on :8080...")
	fmt.Println("Try: curl -X POST http://localhost:8080/v1/createcluster -d '{\"clusterName\":\"test\"}'")
	
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// 実際のHTTPリクエスト例を示すための関数
func demonstrateHTTPUsage() {
	// curl コマンドの例:
	
	// 1. CreateCluster
	// curl -X POST http://localhost:8080/v1/createcluster \
	//   -H "Content-Type: application/x-amz-json-1.1" \
	//   -d '{"clusterName": "my-cluster"}'
	
	// 2. ListClusters
	// curl -X POST http://localhost:8080/v1/listclusters \
	//   -H "Content-Type: application/x-amz-json-1.1" \
	//   -d '{}'
	
	// 3. RunTask
	// curl -X POST http://localhost:8080/v1/runtask \
	//   -H "Content-Type: application/x-amz-json-1.1" \
	//   -d '{"taskDefinition": "my-task:1", "cluster": "my-cluster"}'
}

// 生成されたインターfaces使用例
func demonstrateInterfaceUsage() {
	// 生成されたECSServiceInterfaceを実装したカスタムサービス
	type CustomECSService struct {
		*generated.ECSService
	}

	// CreateClusterの実装をオーバーライド
	func (s *CustomECSService) CreateCluster(ctx context.Context, req *generated.CreateClusterRequest) (*generated.CreateClusterResponse, error) {
		fmt.Printf("Custom CreateCluster called with: %+v\n", req)
		
		// カスタムロジックをここに実装
		// 例: Kubernetesクラスターとの連携、データベースへの保存など
		
		return &generated.CreateClusterResponse{
			// 実際のレスポンスフィールドは generated/types.go を確認
		}, nil
	}

	// カスタムサービスの使用
	customService := &CustomECSService{
		ECSService: generated.NewECSService(),
	}

	// HTTPハンドラーにカスタムサービスを渡す
	http.HandleFunc("/v1/createcluster", generated.HandleCreateCluster(customService))
}