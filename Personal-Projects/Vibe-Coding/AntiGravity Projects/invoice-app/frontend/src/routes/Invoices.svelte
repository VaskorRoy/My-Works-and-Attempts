<script lang="ts">
    import { Plus, Search, FileText, DownloadCloud, Edit2, Trash2 } from 'lucide-svelte';
    import { GetAllInvoices, DeleteInvoice } from '$wailsjs/go/backend/DB';
    import { onMount } from 'svelte';
    import type { backend } from '$wailsjs/go/models';
    import InvoiceForm from '../lib/components/InvoiceForm.svelte';

    let invoices: backend.Invoice[] = [];
    let loading = true;
    let showForm = false;
    let selectedInvoice: backend.Invoice | null = null;

    let searchQuery = '';
    let filterStatus = 'All';

    $: filteredInvoices = invoices.filter(inv => {
        const vendorMatch = (inv.vendor_name || '').toLowerCase().includes(searchQuery.toLowerCase());
        const taskMatch = (inv.task_of_invoice || '').toLowerCase().includes(searchQuery.toLowerCase());
        const typeMatch = (inv.vendor_type || '').toLowerCase().includes(searchQuery.toLowerCase());
        
        const matchesSearch = vendorMatch || taskMatch || typeMatch;
        const matchesStatus = filterStatus === 'All' ? true : 
                              (filterStatus === 'Paid' ? inv.is_paid : !inv.is_paid);
        
        return matchesSearch && matchesStatus;
    });

    // Reactive cast for template use
    $: invoicesAny = filteredInvoices as any[];

    function getSym(currency: string) {
        if (currency === 'BDT') return '৳';
        if (currency === 'EUR') return '€';
        if (currency === 'GBP') return '£';
        return '$';
    }


    async function loadInvoices() {
        loading = true;
        try {
            invoices = await GetAllInvoices() || [];
        } catch (e) {
            console.error(e);
            invoices = [];
        } finally {
            loading = false;
        }
    }

    onMount(loadInvoices);

    async function exportFiltered() {
        if (filteredInvoices.length === 0) { alert("No data to export."); return; }
        try {
            const xlsx = await import('xlsx');
            const flatData = filteredInvoices.map(inv => ({
                'ID': inv.id,
                'Vendor Name': inv.vendor_name,
                'Currency': inv.currency || 'USD',
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
                ...inv.custom_data
            }));
            const ws = xlsx.utils.json_to_sheet(flatData);
            const wb = xlsx.utils.book_new();
            xlsx.utils.book_append_sheet(wb, ws, "Filtered_Invoices");
            xlsx.writeFile(wb, `Filtered_Invoices_${new Date().toISOString().split('T')[0]}.xlsx`);
        } catch(e) { console.error(e); alert("Failed to export view."); }
    }



    function handleEdit(inv: backend.Invoice) {
        selectedInvoice = inv;
        showForm = true;
    }

    async function handleDelete(inv: backend.Invoice) {
        if (!confirm(`Are you sure you want to permanently delete the invoice for ${inv.vendor_name}?`)) return;
        try {
            await DeleteInvoice(inv.id);
            await loadInvoices();
        } catch (e) {
            alert("Failed to delete: " + e);
        }
    }
</script>

<div class="h-full flex flex-col relative">
    {#if showForm}
        <InvoiceForm bind:isOpen={showForm} editInvoice={selectedInvoice} on:saved={loadInvoices} />
    {/if}

    <div class="flex items-center justify-between mb-8">
        <div class="relative flex-1 max-w-md">
            <Search size={18} class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input 
                type="text" 
                bind:value={searchQuery}
                placeholder="Search vendor, type, or task..." 
                class="w-full bg-surface border border-surfaceAcc rounded-xl pl-10 pr-4 py-2.5 focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary transition-all text-sm"
            >
        </div>
        <div class="flex gap-3">
            <select bind:value={filterStatus} class="bg-surface border border-surfaceAcc px-4 py-2.5 rounded-xl text-sm font-medium focus:outline-none focus:border-primary">
                <option value="All">All Statuses</option>
                <option value="Paid">Paid</option>
                <option value="Pending">Pending</option>
            </select>
            
            <button on:click={exportFiltered} class="flex items-center gap-2 bg-surface text-blue-400 border border-surfaceAcc px-4 py-2.5 rounded-xl hover:bg-surfaceAcc/50 transition-colors text-sm font-medium">
                <DownloadCloud size={16} /> Export View
            </button>
            
            <button 
                class="flex items-center gap-2 bg-primary hover:bg-primary/90 text-white px-4 py-2.5 rounded-xl transition-colors text-sm font-medium shadow-lg shadow-primary/20 cursor-pointer"
                on:click={() => { selectedInvoice = null; showForm = true; }}
            >
                <Plus size={18} /> New Invoice
            </button>
        </div>
    </div>

    <!-- Table -->
    <div class="bg-surface rounded-2xl border border-surfaceAcc/50 shadow-sm flex-1 flex flex-col overflow-hidden">
        <div class="overflow-x-auto flex-1">
            <table class="w-full text-left text-sm">
                <thead class="bg-surfaceAcc/20 text-gray-400 text-xs uppercase tracking-wider sticky top-0">
                    <tr>
                        <th class="px-6 py-4 font-medium">Vendor</th>
                        <th class="px-6 py-4 font-medium">Date</th>
                        <th class="px-6 py-4 font-medium">Total</th>
                        <th class="px-6 py-4 font-medium">Net</th>
                        <th class="px-6 py-4 font-medium">Type</th>
                        <th class="px-6 py-4 font-medium">Status</th>
                        <th class="px-6 py-4 text-right">Actions</th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-surfaceAcc/50">
                    {#if loading}
                        <tr><td colspan="7" class="px-6 py-12 text-center text-gray-500">Loading invoices...</td></tr>
                    {:else if invoicesAny.length === 0}
                        <tr>
                            <td colspan="7" class="px-6 py-12 text-center text-gray-500">
                                <div class="flex flex-col items-center gap-2">
                                    <div class="w-12 h-12 bg-surfaceAcc/30 rounded-full flex items-center justify-center text-gray-400 mb-2">
                                        <FileText size={24} />
                                    </div>
                                    No invoices found. Click "New Invoice" to add one.
                                </div>
                            </td>
                        </tr>
                    {:else}
                        {#each invoicesAny as inv}
                            <tr class="hover:bg-surfaceAcc/20 transition-colors">
                                <td class="px-6 py-4 font-medium">{inv.vendor_name || '—'}</td>
                                <td class="px-6 py-4 text-gray-400">{inv.invoice_date || '—'}</td>
                                <td class="px-6 py-4 font-medium">{getSym(inv.currency)}{inv.invoice_value ?? 0}</td>
                                <td class="px-6 py-4 text-primary font-medium">{getSym(inv.currency)}{inv.net_value ?? 0}</td>
                                <td class="px-6 py-4">
                                    <span class="px-2.5 py-1 bg-surfaceAcc rounded-md text-xs">{inv.vendor_type || '—'}</span>
                                </td>
                                <td class="px-6 py-4">
                                    {#if inv.is_paid}
                                        <span class="text-success text-xs font-bold bg-success/10 px-2.5 py-1 rounded-md">Paid</span>
                                    {:else}
                                        <span class="text-secondary text-xs font-bold bg-secondary/10 px-2.5 py-1 rounded-md">Pending</span>
                                    {/if}
                                </td>
                                <td class="px-6 py-4 text-right space-x-2">
                                    <button on:click={() => handleEdit(inv)} class="text-gray-400 hover:text-white transition-colors bg-surfaceAcc/50 hover:bg-primary/50 p-2 rounded-lg" title="Edit Data">
                                        <Edit2 size={16} />
                                    </button>
                                    <button on:click={() => handleDelete(inv)} class="text-gray-400 hover:text-danger transition-colors bg-surfaceAcc/50 hover:bg-danger/20 p-2 rounded-lg" title="Delete Entry">
                                        <Trash2 size={16} />
                                    </button>
                                </td>
                            </tr>
                        {/each}
                    {/if}
                </tbody>
            </table>
        </div>
    </div>
</div>
