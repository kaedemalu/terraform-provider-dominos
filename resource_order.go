package main

import (
	"fmt"
    "log"
    "net/http"
    "net/http/httputil"
    "time"
    "encoding/json"
    "strings"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOrder() *schema.Resource {
	return &schema.Resource{
		Create: resourceOrderCreate,
		Read:   resourceOrderRead,
		Delete: resourceOrderDelete,
		Schema: map[string]*schema.Schema{
            "address_api_object": &schema.Schema{
                Type: schema.TypeString,
                Required: true,
                ForceNew: true,
            },
            "item_codes": &schema.Schema{
                Type: schema.TypeList,
                Required: true,
                Elem: &schema.Schema{
                    Type: schema.TypeString,
                },
                ForceNew: true,
            },
            "store_id": &schema.Schema{
                Type: schema.TypeString,
                Required: true,
                ForceNew: true,
            },
        },
    }
}


func resourceOrderCreate(d *schema.ResourceData, m interface{}) error {
    var client = &http.Client{Timeout: 10 * time.Second}
    store_id := d.Get("store_id").(string)
    address_obj := make(map[string]interface{})
    err := json.NewDecoder(strings.NewReader(d.Get("address_api_object").(string))).Decode(&address_obj)
    if err != nil {
        return err
    }
    order_data := map[string]interface{}{
        "Address": address_obj,
        "Coupons": []string{},
        "CustomerID": "",
        "Extension": "",
        "OrderChannel": "OLO",
        "OrderID": "",
        "NoCombine": true,
        "OrderMethod": "Web",
        "OrderTaker": nil,
        "Payments": []map[string]interface{}{map[string]interface{}{"Type":"Cash"}},
        "Products": []map[string]interface{}{},
        "Market": "",
        "Currency": "",
        "ServiceMethod": "Delivery",
        "Tags": map[string]string{},
        "Version": "1.0",
        "SourceOrganizationURI": "order.dominos.com",
        "LanguageCode": "en",
        "Partners": map[string]string{},
        "NewUser": true,
        "metaData": map[string]string{},
        "Amounts": map[string]string{},
        "BusinessDate": "",
        "EstimatedWaitMinutes": "",
        "PriceOrderTime": "",
        "AmountBreakdown": map[string]string{},
        "StoreID": store_id,
    }

    menuapi, err := getMenuApiObject(fmt.Sprintf("https://order.dominos.com/power/store/%s/menu?lang=en&structured=true", store_id), client)
    if err != nil {
        return err
    }
    codes := d.Get("item_codes").([]interface{})
    for i := range codes {
        variant := menuapi["Variants"].(map[string]interface{})[codes[i].(string)].(map[string]interface{})
        variant["ID"] = 1
        variant["isNew"] = true
        variant["Qty"] = 1
        variant["AutoRemove"] = false
        order_data["Products"] = append(order_data["Products"].([]map[string]interface{}), variant)
    }
    log.Printf("order data: %#v", order_data)
    config := m.(*Config)
    order_data["Email"] = config.EmailAddr
    order_data["FirstName"] = config.FirstName
    order_data["LastName"] = config.LastName
    order_data["Phone"] = config.PhoneNumber
    val_bytes, err := json.Marshal(map[string]interface{}{"Order": order_data})
    if err != nil {
        return err
    }
    val_req, err := http.NewRequest("POST", "https://order.dominos.com/power/price-order", strings.NewReader(string(val_bytes)))
    if err != nil {
        return err
    }
    val_req.Header.Set("Referer", "https://order.dominos.com/en/pages/order/")
    val_req.Header.Set("Content-Type", "application/json")
    dumpreq, err := httputil.DumpRequest(val_req, true)
    if err != nil {
        return err
    }
    log.Printf("http request:  %#v", string(dumpreq))
    val_rsp, err := client.Do(val_req)
    if err != nil {
        return err
    }
    dumprsp, err := httputil.DumpResponse(val_rsp, true)
    if err != nil {
        return err
    }
    log.Printf("http response: %#v", string(dumprsp))
    validate_response_obj := make(map[string]interface{})
    err = json.NewDecoder(val_rsp.Body).Decode(&validate_response_obj)
    if validate_response_obj["Status"].(float64) == -1 {
        return fmt.Errorf("The Dominos API didn't like this request: %#v", validate_response_obj["StatusItems"])
    }
    for k,v := range validate_response_obj["Order"].(map[string]interface{}) {
        if list, ok := v.([]interface{}); !ok || len(list) > 0 {
            order_data[k] = v
        }
    }

    if config.CreditCardNumber != 0 {
        order_data["Payments"] = []map[string]interface{}{map[string]interface{}{
            "Type": "CreditCard",
            "Expiration": config.ExprDate,
            "Amount": order_data["Amounts"].(map[string]interface{})["Customer"],
            "CardType": config.CardType,
            "Number": config.CreditCardNumber,
            "SecurityCode": config.Cvv,
            "PostalCode": config.Zip,
        }}
    }


    order_bytes, err := json.Marshal(map[string]interface{}{"Order": order_data})
    if err != nil {
        return err
    }
    order_req, err := http.NewRequest("POST", "https://order.dominos.com/power/place-order", strings.NewReader(string(order_bytes)))
    if err != nil {
        return err
    }
    order_req.Header.Set("Referer", "https://order.dominos.com/en/pages/order/")
    order_req.Header.Set("Content-Type", "application/json")

    dump_order_req, err := httputil.DumpRequest(order_req, true)
    if err != nil {
        return err
    }
    log.Printf("http request:  %#v", string(dump_order_req))
    order_rsp, err := client.Do(order_req)
    if err != nil {
        return err
    }
    dump_order_rsp, err := httputil.DumpResponse(order_rsp, true)
    if err != nil {
        return err
    }
    log.Printf("http response: %#v", string(dump_order_rsp))
    order_response_obj := make(map[string]interface{})
    err = json.NewDecoder(order_rsp.Body).Decode(&order_response_obj)
    if err != nil {
        return err
    }
    return nil
}

func resourceOrderRead(d *schema.ResourceData, m interface{}) error {
    return nil
}

func resourceOrderDelete(d *schema.ResourceData, m interface{}) error {
    return nil
}