<script lang="ts">
    import { ChartBar, TrendingUp, Clock, FileText, Table, Save, Trash2, Heart, FolderOpen } from 'lucide-svelte';
    import { onMount, onDestroy } from 'svelte';
    import Chart from 'chart.js/auto';
    import * as DB from '$wailsjs/go/backend/DB';
    const { GetAllInvoices, GetChartConfigs, SaveChartConfig, DeleteChartConfig, Notify, GetSetting } = DB as any;
    import { backend } from '$wailsjs/go/models';

    let invoices: backend.Invoice[] = [];
    let canvas: HTMLCanvasElement;
    let chartInstance: Chart;

    // Saved Charts
    let savedConfigs: any[] = [];
    let newConfigName = '';

    // Custom Analytics Graph Controls
    let chartType = 'bar';
    let groupBy = 'cost_type';
    let metricField = 'invoice_value';

    // Derived Stats
    let totalOutput = 0;
    let pendingInvoices = 0;
    let overdueInvoices = 0;
    let averageTAT = 0; // Turn Around Time (Recommendation - Bill Received)

    // Settings
    let exportPath = '';

    onMount(async () => {
        try {
            invoices = await GetAllInvoices() || [];
            savedConfigs = await GetChartConfigs() || [];
            exportPath = await GetSetting('export_path') || '';
            calculateStats();
        } catch(e) { console.error(e) }
    });

    onDestroy(() => {
        if (chartInstance) chartInstance.destroy();
    });

    function calculateStats() {
        totalOutput = invoices.reduce((sum, inv) => sum + (inv.invoice_value ?? 0), 0);
        pendingInvoices = invoices.filter(inv => !inv.is_paid).length;
        
        const now = new Date();
        overdueInvoices = invoices.filter(inv => {
            if (inv.is_paid) return false;
            if (!inv.recommend_date) return false;
            return new Date(inv.recommend_date) < now;
        }).length;
        
        let validTATs = 0;
        let totalTATDays = 0;
        
        invoices.forEach(inv => {
            if (inv.bill_received_date && inv.recommend_date) {
                const received = new Date(inv.bill_received_date);
                const recommended = new Date(inv.recommend_date);
                if (!isNaN(received.getTime()) && !isNaN(recommended.getTime())) {
                    const diffTime = Math.abs(recommended.getTime() - received.getTime());
                    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
                    totalTATDays += diffDays;
                    validTATs++;
                }
            }
        });
        
        averageTAT = validTATs > 0 ? (totalTATDays / validTATs) : 0;
    }

    $: if (invoices && canvas && chartType && groupBy && metricField) {
        setTimeout(generateChart, 50); 
    }

    function generateChart() {
        if (!canvas) return;

        // Group data based on user selection
        const aggr: Record<string, number> = {};
        invoices.forEach(inv => {
            let key = 'Other';
            if (groupBy === 'cost_type') key = inv.cost_type || 'Uncategorized';
            if (groupBy === 'vendor_type') key = inv.vendor_type || 'Uncategorized';
            if (groupBy === 'status') key = inv.is_paid ? 'Paid' : 'Pending';
            if (groupBy === 'month') {
                if (inv.invoice_date) {
                    key = inv.invoice_date.substring(0, 7); // YYYY-MM
                } else key = 'Unknown';
            }
            if (groupBy === 'currency') key = inv.currency || 'USD';

            let val = 1; // for count
            if (metricField === 'invoice_value') val = inv.invoice_value ?? 0;
            if (metricField === 'net_value') val = inv.net_value ?? 0;
            if (metricField === 'vat') val = inv.vat ?? 0;

            aggr[key] = (aggr[key] || 0) + val;
        });

        const labels = Object.keys(aggr);
        const data = Object.values(aggr);

        const isPie = chartType === 'pie' || chartType === 'doughnut';
        
        let bgColors = isPie ? [
            '#3b82f6', '#f97316', '#10b981', '#ef4444', '#8b5cf6', '#06b6d4', '#f43f5e', '#64748b'
        ] : 'rgba(59, 130, 246, 0.8)'; // Tailwind Primary Blue
        
        const metricNames: Record<string, string> = {
            'invoice_value': 'Total Gross', 'net_value': 'Total Net', 'vat': 'Total VAT', 'count': 'Invoice Count'
        };

        if (chartInstance) chartInstance.destroy();

        chartInstance = new Chart(canvas, {
            type: chartType as any,
            data: {
                labels: labels.length ? labels : ['No Data'],
                datasets: [{
                    label: metricNames[metricField] || metricField,
                    data: data.length ? data : [0],
                    backgroundColor: bgColors,
                    borderRadius: isPie ? 0 : 6,
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: { display: isPie }
                },
                scales: isPie ? {} : {
                    y: {
                        beginAtZero: true,
                        grid: { color: 'rgba(100, 116, 139, 0.1)' },
                        ticks: { color: '#64748b' }
                    },
                    x: {
                        grid: { display: false },
                        ticks: { color: '#64748b' }
                    }
                }
            }
        });
    }

    // Export Logic — lazy-loaded to keep initial bundle small
    
    async function exportExcel() {
        if (invoices.length === 0) return alert("No data to export.");
        
        const XLSX = await import('xlsx');
        
        const flatData = invoices.map((inv: any) => {
            const base: Record<string, any> = {
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
            };
            if (inv.custom_data) {
                for (const [key, value] of Object.entries(inv.custom_data)) {
                    base[`Custom: ${key}`] = value;
                }
            }
            return base;
        });

        const ws = XLSX.utils.json_to_sheet(flatData);
        const wb = XLSX.utils.book_new();
        XLSX.utils.book_append_sheet(wb, ws, "Invoices");
        XLSX.writeFile(wb, "Invoices_Export.xlsx");
    }

    async function exportPDF() {
        if (invoices.length === 0) return alert("No data to export.");
        const { jsPDF } = await import('jspdf');
        const { default: autoTable } = await import('jspdf-autotable');
        const doc = new jsPDF();
        
        doc.setFontSize(20);
        doc.setTextColor(30, 60, 150);
        doc.text("Invoice Tracker - Financial Report", 14, 22);
        
        doc.setFontSize(10);
        doc.setTextColor(100);
        doc.text(`Generated: ${new Date().toLocaleString()}`, 14, 30);
        doc.text(`Total Invoices: ${invoices.length} | Overdue: ${overdueInvoices}`, 14, 36);

        const tableData = invoices.map(inv => [
            inv.id,
            inv.vendor_name,
            `${inv.currency} ${inv.invoice_value.toLocaleString()}`,
            inv.invoice_date,
            inv.is_paid ? "Paid" : "Pending"
        ]);

        autoTable(doc, {
            startY: 45,
            head: [['ID', 'Vendor', 'Amount (Gross)', 'Date', 'Status']],
            body: tableData,
            theme: 'striped',
            headStyles: { fillColor: [79, 139, 249] as any }, // Using RGB array for consistency with brand color
            styles: { fontSize: 8 }
        });

        if (chartInstance && canvas) {
            const finalY = (doc as any).lastAutoTable.finalY + 10;
            const imgData = canvas.toDataURL('image/png', 1.0);
            doc.text("Spend Distribution:", 14, finalY);
            doc.addImage(imgData, 'PNG', 14, finalY + 5, 180, 90);
        }

        const fileName = `Report_${Date.now()}.pdf`;
        doc.save(fileName);
        Notify("Export Complete", `PDF report saved as ${fileName}`);
    }

    async function exportPPTX() {
        if (invoices.length === 0) return alert("No data to export.");
        const { default: pptxgen } = await import('pptxgenjs');
        const pres = new pptxgen();

        // Title Slide
        let slide = pres.addSlide();
        slide.addText("Invoice Analytics Summary", { x: 0.5, y: 1.5, w: 9, h: 1, fontSize: 36, color: '363636', align: 'center' });
        slide.addText(`Produced: ${new Date().toLocaleDateString()}`, { x: 0.5, y: 2.5, w: 9, h: 1, fontSize: 18, color: '666666', align: 'center' });

        // Metrics Slide
        let metricsSlide = pres.addSlide();
        metricsSlide.addText("Key Performance Indicators", { x: 0.5, y: 0.5, w: 9, h: 0.5, fontSize: 24, fontFace: 'Arial' });
        metricsSlide.addTable([
            [{ text: "Metric", options: { bold: true } }, { text: "Value", options: { bold: true } }],
            [{ text: "Total Gross Spend" }, { text: `$${totalOutput.toLocaleString()}` }],
            [{ text: "Avg Turn-Around Time" }, { text: `${averageTAT.toFixed(1)} Days` }],
            [{ text: "Pending Invoices" }, { text: pendingInvoices.toString() }],
            [{ text: "Overdue Count" }, { text: overdueInvoices.toString() }]
        ], { x: 0.5, y: 1.5, w: 9, border: { type: 'solid', pt: 1, color: 'E2E8F0' } });

        // Chart Slide
        if (chartInstance && canvas) {
            let chartSlide = pres.addSlide();
            chartSlide.addText("Spend Breakdown Visualization", { x: 0.5, y: 0.5, w: 9, h: 0.5, fontSize: 24 });
            const imgData = canvas.toDataURL('image/png', 1.0);
            chartSlide.addImage({ data: imgData, x: 0.5, y: 1.2, w: 9, h: 4.5 });
        }

        const fileName = `Presentation_${Date.now()}.pptx`;
        pres.writeFile({ fileName });
        Notify("Export Complete", `PowerPoint file generated as ${fileName}`);
    }

    async function saveCurrentChart() {
        if (!newConfigName.trim()) return alert("Please enter a name for this config.");
        try {
            await SaveChartConfig({
                id: 0,
                name: newConfigName.trim(),
                type: chartType,
                group_by: groupBy,
                metric: metricField,
                is_preset: false
            });
            newConfigName = '';
            savedConfigs = await GetChartConfigs() || [];
        } catch(e) { alert(e) }
    }

    function loadSavedChart(c: any) {
        chartType = c.type;
        groupBy = c.group_by;
        metricField = c.metric;
    }

    async function deleteChart(id: number) {
        if (!confirm("Delete this saved config?")) return;
        try {
            await DeleteChartConfig(id);
            savedConfigs = await GetChartConfigs() || [];
        } catch(e) { alert(e) }
    }

</script>

<div class="space-y-6">
    <div class="flex justify-between items-end mb-2">
        <div>
            <h2 class="text-2xl font-bold tracking-tight">Overview metrics</h2>
            <p class="text-sm text-gray-400 mt-1">Snapshot of your invoice performance.</p>
        </div>
        <div class="flex gap-3">
            <button on:click={exportExcel} class="flex items-center gap-2 bg-background border border-surfaceAcc hover:border-primary hover:text-primary transition-colors px-4 py-2 rounded-xl text-sm font-medium">
                <Table size={16} /> Excel
            </button>
            <button on:click={exportPDF} class="flex items-center gap-2 bg-background border border-surfaceAcc hover:border-danger hover:text-danger transition-colors px-4 py-2 rounded-xl text-sm font-medium">
                <FileText size={16} /> Print PDF
            </button>
        </div>
    </div>
    <!-- Metrics Cards Grid -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-6">
        
        <div class="bg-surface rounded-2xl p-6 border border-surfaceAcc/50 shadow-sm flex items-center gap-4 hover:border-surfaceAcc transition-colors">
            <div class="bg-success/10 text-success w-12 h-12 rounded-xl flex items-center justify-center">
                <TrendingUp size={24} />
            </div>
            <div>
                <div class="text-sm text-gray-400 font-medium">Total Output</div>
                <div class="text-2xl font-bold mt-1">${totalOutput.toLocaleString('en-US', {minimumFractionDigits: 2})}</div>
            </div>
        </div>

        <div class="bg-surface rounded-2xl p-6 border border-surfaceAcc/50 shadow-sm flex items-center gap-4 hover:border-surfaceAcc transition-colors">
            <div class="bg-secondary/10 text-secondary w-12 h-12 rounded-xl flex items-center justify-center">
                <Clock size={24} />
            </div>
            <div>
                <div class="text-sm text-gray-400 font-medium">Pending Invoices</div>
                <div class="text-2xl font-bold mt-1">{pendingInvoices}</div>
            </div>
        </div>

        <div class="bg-surface rounded-2xl p-6 border border-surfaceAcc/50 shadow-sm flex items-center gap-4 hover:border-surfaceAcc transition-colors">
            <div class="bg-primary/10 text-primary w-12 h-12 rounded-xl flex items-center justify-center">
                <ChartBar size={24} />
            </div>
            <div>
                <div class="text-sm text-gray-400 font-medium">Avg TAT (Days)</div>
                <div class="text-2xl font-bold mt-1">{averageTAT.toFixed(1)}</div>
            </div>
        </div>

        <div class="bg-surface rounded-2xl p-6 border {overdueInvoices > 0 ? 'border-danger/50 shadow-danger/10' : 'border-surfaceAcc/50'} shadow-sm flex items-center gap-4 transition-colors">
            <div class="bg-danger/10 text-danger w-12 h-12 rounded-xl flex items-center justify-center">
                <Clock size={24} />
            </div>
            <div>
                <div class="text-sm {overdueInvoices > 0 ? 'text-danger' : 'text-gray-400'} font-medium">Overdue (Aging)</div>
                <div class="text-2xl font-bold mt-1 {overdueInvoices > 0 ? 'text-danger' : ''}">{overdueInvoices}</div>
            </div>
        </div>
        
    </div>

    <!-- Analytics Section -->
    <div class="grid grid-cols-1 lg:grid-cols-4 gap-8 mt-8">
        
        <!-- Chart Controls -->
        <div class="lg:col-span-1 bg-surface rounded-2xl border border-surfaceAcc/50 shadow-sm p-6 space-y-6">
            <h3 class="text-lg font-bold">Chart Builder</h3>
            
            <div class="space-y-4">
                <div>
                    <label for="chartTypeSelect" class="block text-xs font-bold text-gray-400 uppercase mb-1.5">Type</label>
                    <select id="chartTypeSelect" bind:value={chartType} class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none">
                        <option value="bar">Bar</option>
                        <option value="line">Line</option>
                        <option value="pie">Pie</option>
                        <option value="doughnut">Doughnut</option>
                    </select>
                </div>

                <div>
                    <label for="groupBySelect" class="block text-xs font-bold text-gray-400 uppercase mb-1.5">Group By</label>
                    <select id="groupBySelect" bind:value={groupBy} class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none">
                        <option value="cost_type">By Cost Type</option>
                        <option value="vendor_type">By Vendor Type</option>
                        <option value="status">By Status</option>
                        <option value="month">By Month</option>
                        <option value="currency">By Currency</option>
                    </select>
                </div>

                <div>
                    <label for="metricSelect" class="block text-xs font-bold text-gray-400 uppercase mb-1.5">Metric</label>
                    <select id="metricSelect" bind:value={metricField} class="w-full bg-background border border-surfaceAcc rounded-lg px-3 py-2 text-sm focus:border-primary focus:outline-none">
                        <option value="invoice_value">Total Gross</option>
                        <option value="net_value">Total Net</option>
                        <option value="vat">Total VAT</option>
                        <option value="count">Count</option>
                    </select>
                </div>

                <hr class="border-surfaceAcc/50" />

                <div>
                    <label for="saveConfigInput" class="block text-xs font-bold text-gray-400 uppercase mb-1.5">Save Configuration</label>
                    <div class="flex gap-2">
                        <input id="saveConfigInput" bind:value={newConfigName} type="text" placeholder="Name..." class="flex-1 bg-background border border-surfaceAcc rounded-lg px-2 py-1.5 text-xs focus:border-primary focus:outline-none">
                        <button on:click={saveCurrentChart} class="bg-primary hover:bg-primary/90 text-white p-1.5 rounded-lg transition-colors">
                            <Save size={16} />
                        </button>
                    </div>
                </div>
            </div>

            <div class="pt-4 border-t border-surfaceAcc/50">
                <h4 class="text-xs font-bold text-gray-400 uppercase mb-4 flex items-center gap-2">
                    <Heart size={14} class="text-danger" /> Saved Views
                </h4>
                <div class="space-y-2 max-h-[300px] overflow-y-auto pr-1 custom-scrollbar">
                    {#each savedConfigs as cfg}
                        <!-- svelte-ignore a11y-no-static-element-interactions -->
                        <!-- svelte-ignore a11y-click-events-have-key-events -->
                        <div class="group flex items-center justify-between bg-background border border-surfaceAcc/30 p-2.5 rounded-xl hover:border-primary transition-colors cursor-pointer" on:click={() => loadSavedChart(cfg)}>
                            <div class="text-sm font-medium truncate w-[70%]">{cfg.name}</div>
                            <button on:click|stopPropagation={() => deleteChart(cfg.id)} class="text-gray-500 hover:text-danger opacity-0 group-hover:opacity-100 transition-opacity p-1">
                                <Trash2 size={14} />
                            </button>
                        </div>
                    {:else}
                        <div class="text-xs text-gray-500 italic text-center py-4">No saved charts.</div>
                    {/each}
                </div>
            </div>
        </div>

        <!-- Canvas and Export Actions -->
        <div class="lg:col-span-3 space-y-4">
            <div class="bg-surface rounded-2xl border border-surfaceAcc/50 shadow-sm p-6 relative">
                 <div class="absolute top-6 right-6 flex gap-2">
                    <button on:click={exportPPTX} class="flex items-center gap-2 bg-background border border-surfaceAcc hover:border-orange-500 hover:text-orange-500 transition-colors px-3 py-1.5 rounded-lg text-xs font-medium">
                        <FileText size={14} /> PPTX
                    </button>
                 </div>
                <div class="h-[450px] w-full">
                    <canvas bind:this={canvas}></canvas>
                </div>
            </div>
            
            <div class="bg-blue-500/5 border border-blue-500/20 rounded-xl p-4 flex items-center justify-between">
                <div class="flex items-center gap-3">
                    <FolderOpen size={18} class="text-blue-400" />
                    <span class="text-sm text-gray-400">Export Location: <span class="text-blue-200">{exportPath || 'Default Downloads'}</span></span>
                </div>
                <button class="text-xs text-blue-400 hover:underline font-medium">Change in Settings</button>
            </div>
        </div>
    </div>
</div>
