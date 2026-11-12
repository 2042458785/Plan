package http_model

type ProductView struct {
	Product
	CategoryName string   `json:"category"`
	Stock        int      `json:"stock"`
	Options      []string `json:"options"`
	Colors       []string `json:"colors"`
}

func ViewProduct(p Product) ProductView {
	v := ProductView{Product: p, CategoryName: p.Category.Name, Options: []string{}, Colors: []string{}}
	v.SKUs = []ProductSKU{}
	options := map[string]bool{}
	colors := map[string]bool{}
	for _, sku := range p.SKUs {
		if !sku.Active {
			continue
		}
		v.SKUs = append(v.SKUs, sku)
		v.Stock += sku.Stock
		if !options[sku.Variant] {
			v.Options = append(v.Options, sku.Variant)
			options[sku.Variant] = true
		}
		if !colors[sku.Color] {
			v.Colors = append(v.Colors, sku.Color)
			colors[sku.Color] = true
		}
	}
	return v
}

type CategoryInput struct {
	Name string `json:"name"`
}

type ProductInput struct {
	CategoryID  uint       `json:"categoryId"`
	ID          string     `json:"id"`
	Category    string     `json:"category"`
	Name        string     `json:"name"`
	English     string     `json:"english"`
	Description string     `json:"description"`
	Price       int64      `json:"price" binding:"cents"`
	Image       string     `json:"image"`
	Available   bool       `json:"available"`
	Material    string     `json:"material"`
	Details     string     `json:"details"`
	Version     uint64     `json:"version"`
	SKUs        []SKUInput `json:"skus"`
}
type SKUInput struct {
	ID      uint   `json:"id"`
	Variant string `json:"variant"`
	Color   string `json:"color"`
	Stock   int    `json:"stock"`
}
type CreateOrderInput struct {
	CustomerID  string           `json:"-"`
	Items       []OrderLineInput `json:"items" binding:"required,min=1,max=30,dive"`
	Note        string           `json:"note"`
	Customer    string           `json:"customer"`
	Contact     string           `json:"contact"`
	Pickup      string           `json:"pickup"`
	Key         string           `json:"key"`
	Fulfillment string           `json:"fulfillment,omitempty"`
	Address     string           `json:"address,omitempty"`
}
type OrderLineInput struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
	Variant   string `json:"variant"`
	Color     string `json:"color"`
}
type UpdateOrderInput struct {
	Status  string `json:"status"`
	Paid    *bool  `json:"paid"`
	Version uint64 `json:"version"`
}
type LoginInput struct {
	Username string `json:"username" binding:"required,max=60"`
	Password string `json:"password" binding:"required,max=200"`
}
type OrderView struct {
	Order
	Token string `json:"token,omitempty"`
}
