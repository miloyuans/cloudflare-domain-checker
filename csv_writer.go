package main

import (
	"encoding/csv"
	"os"
	"reflect"
)

// WriteToCSV 将一个 DomainInfo 结构体切片写入到指定的CSV文件中。
func WriteToCSV(filename string, data []DomainInfo) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建CSV写入器，使用UTF-8编码。
	// Go的csv包默认写入UTF-8，兼容性良好。
	writer := csv.NewWriter(file)
	writer.Comma = ',' // 确保使用逗号作为分隔符

	// 写入CSV头部，根据DomainInfo结构体的`csv`标签生成。
	headers := getCSVHeaders(DomainInfo{})
	if err := writer.Write(headers); err != nil {
		return err
	}

	// 遍历数据并写入每一行
	for _, row := range data {
		record := []string{
			row.AccountName,
			row.Domain,
			row.DomainStatus,
			row.DNSRecordName,
			row.DNSRecordType,
			row.DNSRecordContent,
			row.Notes,
			row.ProxyStatus,
			row.TLSEncryptionMode,
			row.DomainNSInfo,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	writer.Flush() // 确保所有缓冲数据都写入文件
	return writer.Error()
}

// getCSVHeaders 从结构体标签中提取CSV头部名称。
func getCSVHeaders(s interface{}) []string {
	var headers []string
	val := reflect.ValueOf(s)
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if csvTag := field.Tag.Get("csv"); csvTag != "" {
			headers = append(headers, csvTag)
		} else {
			// 如果没有"csv"标签，则使用字段名作为头部
			headers = append(headers, field.Name)
		}
	}
	return headers
}
