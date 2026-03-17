export namespace backend {
	
	export class ChartConfig {
	    id: number;
	    name: string;
	    type: string;
	    group_by: string;
	    metric: string;
	    is_preset: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ChartConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.group_by = source["group_by"];
	        this.metric = source["metric"];
	        this.is_preset = source["is_preset"];
	    }
	}
	export class Invoice {
	    id: number;
	    vendor_name: string;
	    invoice_value: number;
	    invoice_date: string;
	    bill_received_date: string;
	    task_of_invoice: string;
	    vendor_type: string;
	    cost_type: string;
	    recommend_date: string;
	    disbursement_date: string;
	    is_paid: boolean;
	    net_value: number;
	    ait: number;
	    vat: number;
	    ait_percentage: number;
	    vat_percentage: number;
	    currency: string;
	    attachment_path: string;
	    custom_data: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Invoice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.vendor_name = source["vendor_name"];
	        this.invoice_value = source["invoice_value"];
	        this.invoice_date = source["invoice_date"];
	        this.bill_received_date = source["bill_received_date"];
	        this.task_of_invoice = source["task_of_invoice"];
	        this.vendor_type = source["vendor_type"];
	        this.cost_type = source["cost_type"];
	        this.recommend_date = source["recommend_date"];
	        this.disbursement_date = source["disbursement_date"];
	        this.is_paid = source["is_paid"];
	        this.net_value = source["net_value"];
	        this.ait = source["ait"];
	        this.vat = source["vat"];
	        this.ait_percentage = source["ait_percentage"];
	        this.vat_percentage = source["vat_percentage"];
	        this.currency = source["currency"];
	        this.attachment_path = source["attachment_path"];
	        this.custom_data = source["custom_data"];
	    }
	}
	export class Vendor {
	    id: number;
	    name: string;
	    tax_id: string;
	    default_category: string;
	
	    static createFrom(source: any = {}) {
	        return new Vendor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.tax_id = source["tax_id"];
	        this.default_category = source["default_category"];
	    }
	}

}

