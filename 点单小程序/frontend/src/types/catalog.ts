export interface SKU { id:number; productId:string; variant:string; color:string; stock:number; active:boolean; version:number }
export interface Category { id:number; name:string; position:number }
export interface Product { id:string; name:string; english:string; description:string; category:string; categoryId:number; price:number; image:string; gallery?:string[] | null; colorImages?:Record<string,string[]> | null; available:boolean; material:string; details:string; care?:string; version:number; stock:number; options:string[]; colors:string[]; skus:SKU[] }
export interface ShopSetting { quickCover?:string; name:string; announcement:string; location:string; hours:string; open:boolean; version:number }
export interface BagLine { key:string; productId:string; variant:string; color:string; quantity:number }
export interface CustomerAccount { number:string; contact:string; name:string; dorm:string }
