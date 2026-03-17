<script lang="ts">
    import { Database, Plus, Trash2, Tag, CloudUpload, CloudDownload, TriangleAlert, Edit2, Check, X, Shield, Users, FolderOpen } from 'lucide-svelte';
    import { onMount } from 'svelte';
    import * as DBModule from '$wailsjs/go/backend/DB';
    const { GetCustomFields, AddCustomField, DeleteCustomField, GetCategories, AddCategory, DeleteCategory, ClearAllInvoices, AddInvoice, GetAllInvoices, GetVendors, AddVendor, DeleteVendor, GetSetting, UpdateSetting } = DBModule as any;
    import { backend } from '$wailsjs/go/models';

    let customFields: Record<string, string> = {};
    let newFieldName = '';
    let newFieldType = 'TEXT';
    
    let vendorCategories: string[] = [];
    let newVendorCategory = '';
    
    let costCategories: string[] = [];
    let newCostCategory = '';

    let editingCategory: { type: 'vendor' | 'cost' | null, oldName: string, newName: string } = { type: null, oldName: '', newName: '' };

    let vendors: any[] = [];
    let newVendorName = '';
    let newVendorTaxID = '';

    let appPin = localStorage.getItem('app_pin') || '';
    let isPinEditing = false;
    let pinInput = appPin;

    let exportPath = '';
    let exportNotifications = true;

    let loading = true;

    async function loadFields() {
        loading = true;
        try {
            customFields = await GetCustomFields() || {};
            vendorCategories = await GetCategories('vendor') || [];
            costCategories = await GetCategories('cost') || [];
            vendors = await GetVendors() || [];
            exportPath = await GetSetting('export_path') || '';
            const notifSetting = await GetSetting('export_notifications');
            exportNotifications = notifSetting !== 'false';
        } catch(e) { console.error(e) }
        loading = false;
    }

    onMount(loadFields);

    async function addField() {
        if (!newFieldName.trim()) return;
        try {
            await AddCustomField(newFieldName, newFieldType);
            newFieldName = '';
            await loadFields();
        } catch(e) { alert("Failed to add field: " + e) }
    }

    async function deleteField(name: string) {
        if (!confirm(`Delete custom field '${name}'? This affects existing data.`)) return;
        try {
            await DeleteCustomField(name);
            await loadFields();
        } catch(e) { alert("Failed to delete: " + e) }
    }

    async function addCat(type: 'vendor' | 'cost') {
        const val = type === 'vendor' ? newVendorCategory : newCostCategory;
        if (!val.trim()) return;
        try {
            await AddCategory(type, val.trim());
            if (type === 'vendor') newVendorCategory = '';
            else newCostCategory = '';
            await loadFields();
        } catch(e) { alert("Failed to add category: " + e) }
    }

    async function delCat(type: 'vendor' | 'cost', name: string) {
        if (!confirm(`Delete ${type} category '${name}'?`)) return;
        try {
            await DeleteCategory(type, name);
            await loadFields();
        } catch(e) { alert("Failed to delete category: " + e) }
    }

    function startEditCat(type: 'vendor' | 'cost', name: string) {
        editingCategory = { type, oldName: name, newName: name };
    }

    async function saveEditCat() {
        if (!editingCategory.newName.trim() || !editingCategory.type) {
            editingCategory.type = null;
            return;
        }
        try {
            await DBModule.RenameCategory(editingCategory.type, editingCategory.oldName, editingCategory.newName.trim());
            editingCategory = { type: null, oldName: '', newName: '' };
            await loadFields();
        } catch(e) { alert("Failed to rename category: " + e) }
    }

    async function addVendorWhitelist() {
        if (!newVendorName.trim()) return;
        try {
            await AddVendor({ id: 0, name: newVendorName.trim(), tax_id: newVendorTaxID.trim(), default_category: '' });
            newVendorName = '';
            newVendorTaxID = '';
            await loadFields();
        } catch(e) { alert("Failed to add vendor: " + e) }
    }

    async function removeVendorWhitelist(id: number) {
        if (!confirm("Remove this vendor from the whitelist?")) return;
        try {
            await DeleteVendor(id);
            await loadFields();
        } catch(e) { alert("Failed to delete vendor: " + e) }
    }

    async function updateExportSetting(key: string, val: string) {
        try {
            await UpdateSetting(key, val);
        } catch(e) { console.error(e) }
    }

    function savePin() {
        if (pinInput) {
            localStorage.setItem('app_pin', pinInput);
            appPin = pinInput;
        } else {
            localStorage.removeItem('app_pin');
            appPin = '';
        }
        isPinEditing = false;
    }

    async function handleClearDB() {
        if (!confirm("WARNING: This will permanently delete ALL invoices. Are you absolutely sure?")) return;
        try {
            await ClearAllInvoices();
            alert("Database cleared successfully.");
        } catch(e) { alert("Failure: " + e) }
    }

    async function exportBackup() {
        try {
            const invoices = await GetAllInvoices() || [];
            if (invoices.length === 0) { alert("No data to export."); return; }
            
            const xlsx = await import('xlsx');
            const flatData = invoices.map((inv: any) => ({
                'ID': inv.id,
                'Vendor Name': inv.vendor_name,
                'Total (Gross)': inv.invoice_value,
                'Net Value': inv.net_value,
                'AIT Amount': inv.ait,
                'VAT Amount': inv.vat,
                'AIT %': inv.ait_percentage,
                'VAT %': inv.vat_percentage,
                'Status': inv.is_paid ? 'Paid' : 'Pending',
                'Cost Type': inv.cost_type,
                'Vendor Type': inv.vendor_type,
                'Invoice Date': inv.invoice_date,
                'Received': inv.bill_received_date,
                'Recommended': inv.recommend_date,
                'Disbursed': inv.disbursement_date,
                ...inv.custom_data
            }));
            
            const ws = xlsx.utils.json_to_sheet(flatData);
            const wb = xlsx.utils.book_new();
            xlsx.utils.book_append_sheet(wb, ws, "Invoices_Backup");
            
            xlsx.writeFile(wb, `InvoiceBackup_${new Date().toISOString().split('T')[0]}.xlsx`);
        } catch(e) { console.error(e); alert("Failed to export backup."); }
    }

    async function importBackup(e: Event) {
        const file = (e.target as HTMLInputElement).files?.[0];
        if (!file) return;
        
        try {
            loading = true;
            const xlsx = await import('xlsx');
            const buffer = await file.arrayBuffer();
            const wb = xlsx.read(buffer, { type: 'array' });
            const ws = wb.Sheets[wb.SheetNames[0]];
            const data = xlsx.utils.sheet_to_json(ws);
            
            let count = 0;
            for (const row of data as any[]) {
                // Best effort mapping back
                const inv = new backend.Invoice({
                    vendor_name: row['Vendor Name'] || 'Unknown Import',
                    invoice_value: row['Total (Gross)'] || row['Value'] || 0,
                    net_value: row['Net Value'] || 0,
                    ait: row['AIT Amount'] || 0,
                    vat: row['VAT Amount'] || 0,
                    ait_percentage: row['AIT %'] || 0,
                    vat_percentage: row['VAT %'] || 0,
                    is_paid: row['Status'] === 'Paid',
                    cost_type: row['Cost Type'] || 'Operational',
                    vendor_type: row['Vendor Type'] || 'Supplier',
                    invoice_date: row['Invoice Date'] || '',
                    bill_received_date: row['Received'] || '',
                    recommend_date: row['Recommended'] || '',
                    disbursement_date: row['Disbursed'] || '',
                    custom_data: {}
                });
                await AddInvoice(inv);
                count++;
            }
            alert(`Successfully imported ${count} invoices.`);
        } catch(err) {
            alert("Failed to parse/import file: " + err);
        } finally {
            loading = false;
        }
    }
</script>

<div class="max-w-3xl">
    <div class="bg-surface rounded-2xl border border-surfaceAcc/50 shadow-sm p-6 mb-8">
        <div class="flex items-center gap-3 mb-6 pb-6 border-b border-surfaceAcc/50">
            <div class="bg-blue-500/10 text-blue-400 p-2.5 rounded-xl">
                <Database size={24} />
            </div>
            <div>
                <h2 class="text-lg font-bold">Dynamic Schema Configuration</h2>
                <p class="text-sm text-gray-400 mt-1">Add or remove custom fields that will appear on all new invoices.</p>
            </div>
        </div>

        <div class="flex gap-4 items-end mb-8 bg-background p-4 rounded-xl border border-surfaceAcc/30">
            <div class="flex-1">
                <label for="newFieldName" class="block text-xs text-gray-400 font-medium mb-1 uppercase tracking-wider">Field Name</label>
                <input id="newFieldName" bind:value={newFieldName} type="text" placeholder="e.g. Project Code" class="w-full bg-surface border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-primary">
            </div>
            <div class="w-48">
                <label for="newFieldType" class="block text-xs text-gray-400 font-medium mb-1 uppercase tracking-wider">Data Type</label>
                <select id="newFieldType" bind:value={newFieldType} class="w-full bg-surface border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-primary appearance-none">
                    <option value="TEXT">Text</option>
                    <option value="REAL">Number (Decimal)</option>
                    <option value="DATE">Date</option>
                </select>
            </div>
            <button on:click={addField} class="bg-primary hover:bg-primary/90 text-white px-4 py-2 rounded-lg text-sm font-medium flex items-center gap-2 h-[38px] transition-colors">
                <Plus size={16} /> Add Field
            </button>
        </div>

        <div class="space-y-3">
            <h3 class="text-sm font-bold text-gray-300 uppercase tracking-wider mb-4">Active Custom Fields</h3>
            
            {#if loading}
                <div class="text-sm text-gray-500 py-4">Loading fields...</div>
            {:else if Object.keys(customFields).length === 0}
                <div class="text-sm text-gray-500 py-8 text-center italic border-2 border-dashed border-surfaceAcc/50 rounded-xl">No custom fields defined yet.</div>
            {:else}
                {#each Object.entries(customFields) as [name, type]}
                    <div class="flex items-center justify-between p-4 bg-background rounded-xl border border-surfaceAcc/30 group hover:border-surfaceAcc transition-colors">
                        <div>
                            <div class="font-medium text-sm">{name}</div>
                            <div class="text-xs text-primary mt-0.5 font-mono">{type}</div>
                        </div>
                        <button on:click={() => deleteField(name)} class="text-gray-500 hover:text-danger opacity-0 group-hover:opacity-100 transition-all bg-surface p-2 rounded-lg hover:bg-danger/10">
                            <Trash2 size={16} />
                        </button>
                    </div>
                {/each}
            {/if}
        </div>
    </div>

    <!-- Category Settings Row -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-8 mb-8">
        
        <!-- Vendor Categories -->
        <div class="bg-surface rounded-2xl border border-surfaceAcc/50 shadow-sm p-6">
            <div class="flex items-center gap-3 mb-6 pb-6 border-b border-surfaceAcc/50">
                <div class="bg-purple-500/10 text-purple-400 p-2.5 rounded-xl">
                    <Tag size={20} />
                </div>
                <div>
                    <h2 class="text-base font-bold">Vendor Types</h2>
                </div>
            </div>

            <div class="flex gap-2 mb-4">
                <input bind:value={newVendorCategory} type="text" placeholder="New Vendor Type..." class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-primary">
                <button on:click={() => addCat('vendor')} class="bg-surfaceAcc hover:bg-surfaceAcc/70 text-white px-3 py-2 rounded-lg transition-colors">
                    <Plus size={16} />
                </button>
            </div>

            <div class="flex flex-wrap gap-2">
                {#each vendorCategories as cat}
                    {#if editingCategory.type === 'vendor' && editingCategory.oldName === cat}
                        <div class="flex items-center gap-1 bg-background border border-primary px-2 py-1 rounded-md text-xs shadow-md">
                            <!-- svelte-ignore a11y-autofocus -->
                            <input bind:value={editingCategory.newName} class="bg-transparent text-textMain outline-none w-24" on:keydown={(e) => e.key === 'Enter' && saveEditCat()} autofocus/>
                            <button on:click={saveEditCat} class="text-success hover:text-white p-0.5"><Check size={14}/></button>
                            <button on:click={() => editingCategory.type = null} class="text-gray-500 hover:text-danger p-0.5"><X size={14}/></button>
                        </div>
                    {:else}
                        <div class="flex items-center gap-1 bg-background border border-surfaceAcc/50 px-2 py-1 rounded-md text-xs group">
                            {cat}
                            <div class="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                                <button on:click={() => startEditCat('vendor', cat)} class="text-gray-500 hover:text-primary p-0.5" title="Rename"><Edit2 size={12}/></button>
                                <button on:click={() => delCat('vendor', cat)} class="text-gray-500 hover:text-danger p-0.5" title="Delete"><Trash2 size={12}/></button>
                            </div>
                        </div>
                    {/if}
                {/each}
            </div>
        </div>

        <!-- Cost Categories -->
        <div class="bg-surface rounded-2xl border border-surfaceAcc/50 shadow-sm p-6">
            <div class="flex items-center gap-3 mb-6 pb-6 border-b border-surfaceAcc/50">
                <div class="bg-emerald-500/10 text-emerald-400 p-2.5 rounded-xl">
                    <Tag size={20} />
                </div>
                <div>
                    <h2 class="text-base font-bold">Cost Types</h2>
                </div>
            </div>

            <div class="flex gap-2 mb-4">
                <input bind:value={newCostCategory} type="text" placeholder="New Cost Type..." class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-primary">
                <button on:click={() => addCat('cost')} class="bg-surfaceAcc hover:bg-surfaceAcc/70 text-white px-3 py-2 rounded-lg transition-colors">
                    <Plus size={16} />
                </button>
            </div>

            <div class="flex flex-wrap gap-2">
                {#each costCategories as cat}
                    {#if editingCategory.type === 'cost' && editingCategory.oldName === cat}
                        <div class="flex items-center gap-1 bg-background border border-primary px-2 py-1 rounded-md text-xs shadow-md">
                            <!-- svelte-ignore a11y-autofocus -->
                            <input bind:value={editingCategory.newName} class="bg-transparent text-textMain outline-none w-24" on:keydown={(e) => e.key === 'Enter' && saveEditCat()} autofocus/>
                            <button on:click={saveEditCat} class="text-success hover:text-white p-0.5"><Check size={14}/></button>
                            <button on:click={() => editingCategory.type = null} class="text-gray-500 hover:text-danger p-0.5"><X size={14}/></button>
                        </div>
                    {:else}
                        <div class="flex items-center gap-1 bg-background border border-surfaceAcc/50 px-2 py-1 rounded-md text-xs group">
                            {cat}
                            <div class="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                                <button on:click={() => startEditCat('cost', cat)} class="text-gray-500 hover:text-primary p-0.5" title="Rename"><Edit2 size={12}/></button>
                                <button on:click={() => delCat('cost', cat)} class="text-gray-500 hover:text-danger p-0.5" title="Delete"><Trash2 size={12}/></button>
                            </div>
                        </div>
                    {/if}
                {/each}
            </div>
        </div>

    </div>

    <!-- Vendor Book & Security -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-8 mb-8">
        
        <!-- Vendor Whitelist -->
        <div class="bg-surface rounded-2xl border border-surfaceAcc/50 shadow-sm p-6">
            <div class="flex items-center gap-3 mb-6 pb-6 border-b border-surfaceAcc/50">
                <div class="bg-indigo-500/10 text-indigo-400 p-2.5 rounded-xl">
                    <Users size={20} />
                </div>
                <div>
                    <h2 class="text-base font-bold">Vendor Directory</h2>
                    <p class="text-xs text-gray-400 mt-0.5">Whitelist trusted vendors.</p>
                </div>
            </div>

            <div class="flex gap-2 mb-4">
                <input bind:value={newVendorName} type="text" placeholder="Vendor Name..." class="w-1/2 bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-primary">
                <input bind:value={newVendorTaxID} type="text" placeholder="Tax ID (opt)" class="w-1/3 bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-primary">
                <button on:click={addVendorWhitelist} class="bg-primary hover:bg-primary/90 text-white px-3 py-2 rounded-lg transition-colors flex-1 flex justify-center">
                    <Plus size={16} />
                </button>
            </div>

            <div class="space-y-2 mt-4 max-h-[200px] overflow-y-auto pr-2 custom-scrollbar">
                {#each vendors as v}
                    <div class="flex items-center justify-between bg-background border border-surfaceAcc/50 p-2.5 rounded-lg text-sm group">
                        <div>
                            <span class="font-medium">{v.name}</span>
                            {#if v.tax_id}<span class="text-xs text-primary ml-2 bg-primary/10 px-1.5 py-0.5 rounded">ID: {v.tax_id}</span>{/if}
                        </div>
                        <button on:click={() => removeVendorWhitelist(v.id)} class="text-gray-500 hover:text-danger opacity-0 group-hover:opacity-100 transition-opacity">
                            <Trash2 size={14} />
                        </button>
                    </div>
                {:else}
                    <div class="text-xs text-gray-500 italic text-center py-4">No vendors whitelisted yet.</div>
                {/each}
            </div>
        </div>

        <!-- Security / PIN -->
        <div class="bg-surface rounded-2xl border border-surfaceAcc/50 shadow-sm p-6">
            <div class="flex items-center gap-3 mb-6 pb-6 border-b border-surfaceAcc/50">
                <div class="bg-amber-500/10 text-amber-400 p-2.5 rounded-xl">
                    <Shield size={20} />
                </div>
                <div>
                    <h2 class="text-base font-bold">App Security</h2>
                    <p class="text-xs text-gray-400 mt-0.5">Lock application with a Passcode.</p>
                </div>
            </div>

            {#if !isPinEditing}
                <div class="flex items-center justify-between bg-background p-4 rounded-xl border border-surfaceAcc/50">
                    <div>
                        <div class="text-sm font-medium">Application PIN</div>
                        <div class="text-xs text-gray-400 mt-1">{appPin ? 'Enabled (Lock on startup)' : 'Disabled'}</div>
                    </div>
                    <button on:click={() => { isPinEditing = true; pinInput = appPin; }} class="bg-surfaceAcc hover:bg-surfaceAcc/70 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors">
                        {appPin ? 'Change PIN' : 'Setup PIN'}
                    </button>
                </div>
            {:else}
                <div class="bg-background p-4 rounded-xl border border-primary shadow-lg shadow-primary/10">
                    <label for="pin" class="block text-xs font-bold text-gray-400 uppercase mb-2">Enter New PIN</label>
                    <input id="pin" type="password" bind:value={pinInput} placeholder="Leave empty to disable" class="w-full bg-surface border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-primary mb-4 text-center tracking-widest font-mono text-lg">
                    <div class="flex gap-2">
                        <button on:click={() => isPinEditing = false} class="flex-1 bg-surfaceAcc hover:bg-surfaceAcc/70 text-white py-2 rounded-lg text-sm font-medium transition-colors">Cancel</button>
                        <button on:click={savePin} class="flex-1 bg-primary hover:bg-primary/90 text-white py-2 rounded-lg text-sm font-medium transition-colors">Save</button>
                    </div>
                </div>
            {/if}
        </div>

    </div>

    <!-- Export Settings -->
    <div class="bg-surface rounded-2xl border border-surfaceAcc/50 shadow-sm p-6 mb-8 mt-8">
        <div class="flex items-center gap-3 mb-6 pb-6 border-b border-surfaceAcc/50">
            <div class="bg-blue-500/10 text-blue-400 p-2.5 rounded-xl">
                <FolderOpen size={24} />
            </div>
            <div>
                <h2 class="text-lg font-bold">Export Configuration</h2>
                <p class="text-sm text-gray-400 mt-1">Global preferences for reporting and data output.</p>
            </div>
        </div>

        <div class="space-y-6">
            <div>
                <label for="exportPathInput" class="block text-xs font-bold text-gray-400 uppercase mb-2">Default Export Directory</label>
                <div class="flex gap-2">
                    <input id="exportPathInput" type="text" bind:value={exportPath} on:change={() => updateExportSetting('export_path', exportPath)} placeholder="/home/user/Documents/Invoices" class="flex-1 bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none">
                </div>
                <p class="text-[10px] text-gray-500 mt-1">Leave empty to use system default (Downloads).</p>
            </div>

            <div class="flex items-center justify-between bg-background p-4 rounded-xl border border-surfaceAcc/30">
                <div>
                    <div class="text-sm font-medium">Export Completion Notifications</div>
                    <div class="text-xs text-gray-400 mt-1">Show a system popup when report generation finishes.</div>
                </div>
                <label class="relative inline-flex items-center cursor-pointer">
                    <input type="checkbox" bind:checked={exportNotifications} on:change={() => updateExportSetting('export_notifications', String(exportNotifications))} class="sr-only peer">
                    <div class="w-11 h-6 bg-surfaceAcc peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
                </label>
            </div>
        </div>
    </div>

    <!-- Data Management -->
    <div class="bg-surface rounded-2xl border border-danger/20 shadow-sm p-6">
        <div class="flex items-center gap-3 mb-6 pb-6 border-b border-surfaceAcc/50">
            <div class="bg-danger/10 text-danger p-2.5 rounded-xl">
                <TriangleAlert size={24} />
            </div>
            <div>
                <h2 class="text-lg font-bold text-danger">Database Management</h2>
                <p class="text-sm text-gray-400 mt-1">Export, Import, or Clear your exact embedded DB.</p>
            </div>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <button on:click={exportBackup} class="flex items-center justify-center gap-2 bg-background border border-surfaceAcc p-4 rounded-xl hover:bg-surfaceAcc/30 transition-colors group">
                <CloudDownload size={18} class="text-blue-400 group-hover:-translate-y-1 transition-transform" />
                <span class="text-sm font-medium">Export Backup (Excel)</span>
            </button>
            
            <label class="flex items-center justify-center gap-2 bg-background border border-surfaceAcc p-4 rounded-xl hover:bg-surfaceAcc/30 transition-colors group cursor-pointer relative">
                <CloudUpload size={18} class="text-success group-hover:-translate-y-1 transition-transform" />
                <span class="text-sm font-medium">Import from Excel/CSV</span>
                <input type="file" accept=".xlsx,.xls,.csv" on:change={importBackup} class="hidden">
            </label>

            <button on:click={handleClearDB} class="flex items-center justify-center gap-2 bg-danger/10 border border-danger/30 text-danger p-4 rounded-xl hover:bg-danger/20 transition-colors">
                <Trash2 size={18} />
                <span class="text-sm font-medium">Wipe Database</span>
            </button>
        </div>
    </div>
</div>
