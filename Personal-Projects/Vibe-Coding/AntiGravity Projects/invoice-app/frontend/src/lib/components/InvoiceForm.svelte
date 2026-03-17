<script lang="ts">
    import { X, Save } from 'lucide-svelte';
    import OCRDropzone from './OCRDropzone.svelte';
    import * as DB from '$wailsjs/go/backend/DB';
    const { AddInvoice, UpdateInvoice, GetCustomFields, GetCategories, GetVendors } = DB as any;
    import { backend } from '$wailsjs/go/models';
    import { onMount, createEventDispatcher } from 'svelte';

    const dispatch = createEventDispatcher();

    export let isOpen = false;
    export let editInvoice: backend.Invoice | null = null;
    
    // Core Invoice Data
    let invoiceId = 0;
    let vendorName = '';
    let invoiceValue = 0;
    let invoiceDate = '';
    let billReceivedDate = new Date().toISOString().split('T')[0];
    let taskOfInvoice = '';
    let vendorType = 'Supplier';
    let costType = 'Operational';
    let recommendDate = '';
    let disbursementDate = '';
    let isPaid = false;
    let currency = 'USD';
    let attachmentPath = '';

    $: currencySymbol = currency === 'BDT' ? '৳' : 
                       (currency === 'EUR' ? '€' : 
                       (currency === 'GBP' ? '£' : '$'));

    // Tax Breakdowns
    let aitPercentage = 0;
    let vatPercentage = 0;
    let aitAmount = 0;
    let vatAmount = 0;
    let netValue = 0;

    $: if (disbursementDate) {
        // Auto-mark paid when disbursement date is provided
        isPaid = true;
    }

    const round2 = (num: number) => Math.round(num * 100) / 100;

    function recalculateTaxesFromNet() {
        // Enforce 2 decimal places max initially
        netValue = round2(netValue);

        // AIT = {Net / (1 - AIT%)} - Net
        const a = round2(aitPercentage) / 100;
        const v = round2(vatPercentage) / 100;
        
        let baseForVat = netValue;
        if (a < 1 && a > 0) {
            aitAmount = round2((netValue / (1 - a)) - netValue);
            baseForVat = netValue / (1 - a);
        } else {
            aitAmount = 0;
        }

        vatAmount = round2(baseForVat * v);
        invoiceValue = round2(netValue + aitAmount + vatAmount);
    }

    function recalculateNetFromGross() {
        invoiceValue = round2(invoiceValue);
        // Gross = Net * (1+V) / (1-A)  => Net = Gross * (1-A) / (1+V)
        const a = round2(aitPercentage) / 100;
        const v = round2(vatPercentage) / 100;

        // Prevent division by zero or negative logic
        if (a < 1) {
            netValue = round2(invoiceValue * (1 - a) / (1 + v));
            recalculateTaxesFromNet(); // This sets AIT, VAT, and stabilizes invoiceValue
        }
    }

    function manualTaxEdit() {
        aitAmount = round2(aitAmount);
        vatAmount = round2(vatAmount);
        // User manually overrides AIT or VAT completely. We treat Net as fixed, Gross updates.
        invoiceValue = round2(netValue + aitAmount + vatAmount);
    }

    // Custom Fields
    let customFieldsDef: Record<string, string> = {};
    let customData: Record<string, string> = {};

    let vendorCategories: string[] = [];
    let costCategories: string[] = [];
    let whitelistedVendors: any[] = [];

    onMount(async () => {
        try {
            customFieldsDef = await GetCustomFields() || {};
            vendorCategories = await GetCategories('vendor') || [];
            costCategories = await GetCategories('cost') || [];
            whitelistedVendors = await GetVendors() || [];
            
            // Provide fallbacks if DB is empty
            if (vendorCategories.length === 0) vendorCategories = ['Supplier', 'Contractor', 'Software/SaaS', 'Utility', 'Consulting', 'Other'];
            if (costCategories.length === 0) costCategories = ['Operational', 'Capital', 'R&D', 'Marketing', 'Administrative', 'Other'];
        } catch(e) { console.error("Failed to load schema", e) }
    });

    $: if (isOpen && editInvoice) {
        const inv = editInvoice as any;
        invoiceId = inv.id;
        vendorName = inv.vendor_name || '';
        invoiceValue = inv.invoice_value;
        invoiceDate = inv.invoice_date || '';
        billReceivedDate = inv.bill_received_date || '';
        taskOfInvoice = inv.task_of_invoice || '';
        vendorType = inv.vendor_type || '';
        costType = inv.cost_type || '';
        recommendDate = inv.recommend_date || '';
        disbursementDate = inv.disbursement_date || '';
        isPaid = inv.is_paid || false;
        currency = inv.currency || 'USD';
        attachmentPath = inv.attachment_path || '';
        
        aitPercentage = inv.ait_percentage || 0;
        vatPercentage = inv.vat_percentage || 0;
        aitAmount = inv.ait || 0;
        vatAmount = inv.vat || 0;
        netValue = inv.net_value || 0;
        
        customData = inv.custom_data || {};
    }

    function handleOCRData(event: CustomEvent<{text: string, file: File}>) {
        const text = event.detail.text;
        
        // Try finding a date (MM/DD/YYYY or YYYY-MM-DD or DD/MM/YYYY)
        const dateMatch = text.match(/\b(\d{1,4}[-/.]\d{1,2}[-/.]\d{1,4})\b/);
        if (dateMatch) invoiceDate = dateMatch[1];
        
        // Try finding a Total Amount
        const amountMatch = text.match(/(?:total|amount|due|balance).*?\$?\s*([0-9,]+(?:\.\d{2})?)/i);
        if (amountMatch) {
            invoiceValue = parseFloat(amountMatch[1].replace(/,/g, ''));
            recalculateNetFromGross();
        }
        
        // Try finding a Vendor Name (First few lines, skipping obvious header words)
        const lines = text.split('\n').map(l => l.trim()).filter(l => l.length > 2);
        if (lines.length > 0) {
            vendorName = lines.find(l => !l.toLowerCase().includes('invoice') && !l.match(/\d/)) || lines[0];
            vendorName = vendorName.substring(0, 50);
        }
    }

    async function handleSave() {
        if (!vendorName.trim()) {
            alert('Vendor Name is required.');
            return;
        }
        if (invoiceValue <= 0) {
            alert('Invoice Value must be greater than zero.');
            return;
        }

        const inv = new backend.Invoice({
            id: invoiceId,
            vendor_name: vendorName.trim(),
            invoice_value: invoiceValue,
            invoice_date: invoiceDate,
            bill_received_date: billReceivedDate,
            task_of_invoice: taskOfInvoice,
            vendor_type: vendorType,
            cost_type: costType,
            recommend_date: recommendDate,
            disbursement_date: disbursementDate,
            is_paid: isPaid,
            net_value: netValue,
            ait: aitAmount,
            vat: vatAmount,
            ait_percentage: aitPercentage,
            vat_percentage: vatPercentage,
            currency: currency,
            attachment_path: attachmentPath.trim(),
            custom_data: customData
        });

        try {
            if (invoiceId > 0) {
                await UpdateInvoice(inv);
            } else {
                await AddInvoice(inv);
            }
            dispatch('saved');
            close();
        } catch(e) {
            alert("Failed to save invoice: " + e);
        }
    }

    function close() {
        isOpen = false;
        editInvoice = null;
        // Reset all fields to defaults
        invoiceId = 0;
        vendorName = '';
        invoiceValue = 0;
        invoiceDate = '';
        billReceivedDate = new Date().toISOString().split('T')[0];
        taskOfInvoice = '';
        vendorType = 'Supplier';
        costType = 'Operational';
        recommendDate = '';
        disbursementDate = '';
        isPaid = false;
        aitPercentage = 0;
        vatPercentage = 0;
        aitAmount = 0;
        vatAmount = 0;
        netValue = 0;
        currency = 'USD';
        attachmentPath = '';
        customData = {};
    }
</script>

{#if isOpen}
<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <!-- Backdrop -->
    <div class="absolute inset-0 bg-background/80 backdrop-blur-sm" on:click={close}></div>
    
    <!-- Modal -->
    <div class="bg-surface border border-surfaceAcc shadow-2xl rounded-2xl w-full max-w-4xl max-h-[90vh] flex flex-col relative z-10 overflow-hidden">
        
        <!-- Header -->
        <div class="h-16 border-b border-surfaceAcc flex items-center justify-between px-6 bg-surface/50">
            <h2 class="text-xl font-bold">{invoiceId > 0 ? 'Edit Invoice' : 'Record New Invoice'}</h2>
            <button class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white rounded-lg hover:bg-surfaceAcc transition-colors" on:click={close}>
                <X size={20} />
            </button>
        </div>

        <!-- Body -->
        <div class="flex-1 overflow-y-auto p-6 grid grid-cols-1 md:grid-cols-2 gap-8">
            
            <!-- Left Side: OCR Upload -->
            <div class="space-y-4">
                <OCRDropzone on:extracted={handleOCRData} on:reset={() => {}}/>
                
                <div class="bg-background border border-surfaceAcc/50 rounded-xl p-4 text-sm text-gray-400">
                    <span class="text-primary font-bold">Pro Tip:</span> 
                    Upload a high-contrast image of the invoice. The local OCR engine will attempt to pre-fill Vendor Name, Date, and Amount automatically.
                </div>
            </div>

            <!-- Right Side: Manual Form -->
            <div class="space-y-5">
                
                <div class="grid grid-cols-[2fr_1fr_2fr] gap-4">
                    <div>
                        <label for="vendorName" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">Vendor Name <span class="text-danger">*</span></label>
                        <input id="vendorName" list="vendorList" bind:value={vendorName} type="text" class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none placeholder-gray-600" placeholder="Acme Corp">
                        <datalist id="vendorList">
                            {#each whitelistedVendors as v}
                                <option value={v.name}>{v.tax_id ? `Tax ID: ${v.tax_id}` : ''}</option>
                            {/each}
                        </datalist>
                    </div>
                    <div>
                        <label for="currency" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">Currency</label>
                        <select id="currency" bind:value={currency} class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none appearance-none font-medium">
                            <option value="USD">USD ($)</option>
                            <option value="EUR">EUR (€)</option>
                            <option value="GBP">GBP (£)</option>
                            <option value="BDT">BDT (৳)</option>
                        </select>
                    </div>
                    <div>
                        <label for="invoiceValue" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5" title="Auto-calculated from Net + Taxes">Gross Amount <span class="text-danger">*</span></label>
                        <div class="relative">
                            <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 font-medium">{currencySymbol}</span>
                            <input id="invoiceValue" bind:value={invoiceValue} on:input={recalculateNetFromGross} type="number" step="0.01" class="w-full bg-background border border-surfaceAcc rounded-lg pl-8 pr-3 py-2 text-sm focus:border-primary focus:outline-none transition-colors hover:border-secondary focus:border-secondary">
                        </div>
                    </div>
                </div>

                <!-- Tax Breakdown Section -->
                <div class="bg-surfaceAcc/10 border border-surfaceAcc/50 rounded-xl p-4 space-y-4">
                    <div class="text-xs font-bold text-primary uppercase tracking-wider">Tax Calculation</div>
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label for="aitPercentage" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">AIT %</label>
                            <div class="relative">
                                <span class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500">%</span>
                                <input id="aitPercentage" bind:value={aitPercentage} on:input={recalculateTaxesFromNet} type="number" step="0.1" class="w-full bg-background border border-surfaceAcc rounded-lg pl-3 pr-8 py-2 text-sm focus:border-primary focus:outline-none">
                            </div>
                        </div>
                        <div>
                            <label for="vatPercentage" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">VAT %</label>
                            <div class="relative">
                                <span class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500">%</span>
                                <input id="vatPercentage" bind:value={vatPercentage} on:input={recalculateTaxesFromNet} type="number" step="0.1" class="w-full bg-background border border-surfaceAcc rounded-lg pl-3 pr-8 py-2 text-sm focus:border-primary focus:outline-none">
                            </div>
                        </div>
                    </div>
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <label for="aitAmount" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">AIT Amount</label>
                            <div class="relative">
                                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 font-medium">{currencySymbol}</span>
                                <input id="aitAmount" bind:value={aitAmount} on:input={manualTaxEdit} type="number" step="0.01" class="w-full bg-background border border-surfaceAcc rounded-lg pl-8 pr-3 py-2 text-sm focus:border-primary focus:outline-none transition-colors hover:border-secondary focus:border-secondary" title="Auto-computed. Edit to override.">
                            </div>
                        </div>
                        <div>
                            <label for="vatAmount" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">VAT Amount</label>
                            <div class="relative">
                                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 font-medium">{currencySymbol}</span>
                                <input id="vatAmount" bind:value={vatAmount} on:input={manualTaxEdit} type="number" step="0.01" class="w-full bg-background border border-surfaceAcc rounded-lg pl-8 pr-3 py-2 text-sm focus:border-primary focus:outline-none transition-colors hover:border-secondary focus:border-secondary" title="Auto-computed. Edit to override.">
                            </div>
                        </div>
                    </div>
                    <div>
                        <label for="netValue" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">Net Value</label>
                        <div class="relative">
                            <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 font-medium">{currencySymbol}</span>
                            <input id="netValue" bind:value={netValue} on:input={recalculateTaxesFromNet} type="number" step="0.01" class="w-full bg-background border border-surfaceAcc rounded-lg pl-8 pr-3 py-2 text-sm focus:border-primary focus:outline-none transition-colors hover:border-secondary">
                        </div>
                    </div>
                </div>

                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label for="invoiceDate" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">Invoice Date</label>
                        <input id="invoiceDate" bind:value={invoiceDate} type="date" class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none text-gray-300">
                    </div>
                    <div>
                        <label for="billReceivedDate" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">Date Received</label>
                        <input id="billReceivedDate" bind:value={billReceivedDate} type="date" class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none text-gray-300">
                    </div>
                </div>

                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label for="vendorType" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">Vendor Type</label>
                        <select id="vendorType" bind:value={vendorType} class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none appearance-none">
                            {#each vendorCategories as cat}
                                <option value={cat}>{cat}</option>
                            {/each}
                        </select>
                    </div>
                    <div>
                        <label for="costType" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">Cost Type</label>
                        <select id="costType" bind:value={costType} class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none appearance-none">
                            {#each costCategories as cat}
                                <option value={cat}>{cat}</option>
                            {/each}
                        </select>
                    </div>
                </div>

                <div>
                    <label for="taskOfInvoice" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">Task / Project</label>
                    <input id="taskOfInvoice" bind:value={taskOfInvoice} type="text" class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none" placeholder="e.g. Server Upgrades">
                </div>

                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label for="recommendDate" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">Recommendation Date</label>
                        <input id="recommendDate" bind:value={recommendDate} type="date" class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none text-gray-300">
                    </div>
                    <div>
                        <label for="disbursementDate" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">Disbursement Date</label>
                        <input id="disbursementDate" bind:value={disbursementDate} type="date" class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none text-gray-300">
                    </div>
                </div>

                <div>
                    <label for="attachmentPath" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">Invoice PDF Path</label>
                    <input id="attachmentPath" bind:value={attachmentPath} type="text" class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm text-gray-400 font-mono focus:border-primary focus:outline-none placeholder-gray-700" placeholder="/absolute/path/to/local/invoice.pdf">
                </div>

                <div class="flex items-center gap-2 pt-2">
                    <input id="isPaid" type="checkbox" bind:checked={isPaid} class="w-4 h-4 rounded border-surfaceAcc bg-background text-primary focus:ring-primary focus:ring-offset-surface">
                    <label for="isPaid" class="text-sm font-medium text-gray-300">Mark as Paid / Payment Completed</label>
                </div>

                <hr class="border-surfaceAcc/50" />

                <!-- Dynamic Custom Fields from DB Schema -->
                {#if Object.keys(customFieldsDef).length > 0}
                    <div class="space-y-4">
                        <div class="text-xs font-bold text-primary uppercase tracking-wider">Custom Fields</div>
                        {#each Object.entries(customFieldsDef) as [name, type]}
                            <div>
                                <label for="custom-{name}" class="block text-xs font-bold text-gray-400 uppercase tracking-wider mb-1.5">{name}</label>
                                {#if type === 'TEXT'}
                                    <input id="custom-{name}" bind:value={customData[name]} type="text" class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none">
                                {:else if type === 'DATE'}
                                    <input id="custom-{name}" bind:value={customData[name]} type="date" class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none">
                                {:else}
                                    <input id="custom-{name}" bind:value={customData[name]} type="number" class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none">
                                {/if}
                            </div>
                        {/each}
                    </div>
                {/if}

            </div>
        </div>

        <!-- Footer -->
        <div class="p-6 border-t border-surfaceAcc bg-surface/50 flex justify-end gap-3">
            <button on:click={close} class="px-5 py-2.5 rounded-xl font-medium text-sm hover:bg-surfaceAcc transition-colors">
                Cancel
            </button>
            <button on:click={handleSave} class="px-5 py-2.5 rounded-xl font-medium text-sm bg-primary hover:bg-primary/90 text-white flex items-center gap-2 shadow-lg shadow-primary/20 transition-all">
                <Save size={18} /> Save Invoice
            </button>
        </div>

    </div>
</div>
{/if}
