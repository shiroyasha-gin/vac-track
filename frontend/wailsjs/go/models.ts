export namespace app {
	
	export class ActionResult {
	    ok: boolean;
	    code: string;
	    field: string;
	    message: string;
	    entityId: number;
	
	    static createFrom(source: any = {}) {
	        return new ActionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.code = source["code"];
	        this.field = source["field"];
	        this.message = source["message"];
	        this.entityId = source["entityId"];
	    }
	}
	export class AdjustmentDTO {
	    id: number;
	    vaccineId: number;
	    vaccineName: string;
	    unit: string;
	    amount: number;
	    balanceBefore: number;
	    balanceAfter: number;
	    reason: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AdjustmentDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.vaccineId = source["vaccineId"];
	        this.vaccineName = source["vaccineName"];
	        this.unit = source["unit"];
	        this.amount = source["amount"];
	        this.balanceBefore = source["balanceBefore"];
	        this.balanceAfter = source["balanceAfter"];
	        this.reason = source["reason"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class AdjustmentInputDTO {
	    vaccineId: number;
	    month: string;
	    amount: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new AdjustmentInputDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vaccineId = source["vaccineId"];
	        this.month = source["month"];
	        this.amount = source["amount"];
	        this.reason = source["reason"];
	    }
	}
	export class AuditEventDTO {
	    id: number;
	    eventType: string;
	    entityType: string;
	    entityId: string;
	    month: string;
	    reason: string;
	    metadata: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditEventDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.eventType = source["eventType"];
	        this.entityType = source["entityType"];
	        this.entityId = source["entityId"];
	        this.month = source["month"];
	        this.reason = source["reason"];
	        this.metadata = source["metadata"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class AuditFilterDTO {
	    eventType: string;
	    month: string;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new AuditFilterDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.eventType = source["eventType"];
	        this.month = source["month"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}
	export class AuditPageDTO {
	    events: AuditEventDTO[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new AuditPageDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.events = this.convertValues(source["events"], AuditEventDTO);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackupResultDTO {
	    ok: boolean;
	    code: string;
	    message: string;
	    path: string;
	    sizeBytes: number;
	    sha256: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.code = source["code"];
	        this.message = source["message"];
	        this.path = source["path"];
	        this.sizeBytes = source["sizeBytes"];
	        this.sha256 = source["sha256"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class BootstrapInfo {
	    ready: boolean;
	    fatalError: string;
	    dataDir: string;
	    databasePath: string;
	    version: string;
	    today: string;
	    currentMonth: string;
	    lowStockThreshold: number;
	    publisherEndpoint: string;
	    publisherTokenConfigured: boolean;
	    publisherTokenStorage: string;
	
	    static createFrom(source: any = {}) {
	        return new BootstrapInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.fatalError = source["fatalError"];
	        this.dataDir = source["dataDir"];
	        this.databasePath = source["databasePath"];
	        this.version = source["version"];
	        this.today = source["today"];
	        this.currentMonth = source["currentMonth"];
	        this.lowStockThreshold = source["lowStockThreshold"];
	        this.publisherEndpoint = source["publisherEndpoint"];
	        this.publisherTokenConfigured = source["publisherTokenConfigured"];
	        this.publisherTokenStorage = source["publisherTokenStorage"];
	    }
	}
	export class CreateUsageInputDTO {
	    patientId: number;
	    vaccineId: number;
	    administeredOn: string;
	    quantity: number;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateUsageInputDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.patientId = source["patientId"];
	        this.vaccineId = source["vaccineId"];
	        this.administeredOn = source["administeredOn"];
	        this.quantity = source["quantity"];
	        this.note = source["note"];
	    }
	}
	export class DailyTotalDTO {
	    date: string;
	    day: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new DailyTotalDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.day = source["day"];
	        this.total = source["total"];
	    }
	}
	export class ExportResultDTO {
	    ok: boolean;
	    code: string;
	    message: string;
	    path: string;
	    bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.code = source["code"];
	        this.message = source["message"];
	        this.path = source["path"];
	        this.bytes = source["bytes"];
	    }
	}
	export class ForecastRowDTO {
	    vaccineId: number;
	    name: string;
	    code: string;
	    unit: string;
	    initialized: boolean;
	    method: string;
	    sampleMonths: string[];
	    sampleCount: number;
	    estimatedDemand: number;
	    openingSuggestion: number;
	    priorMonth: string;
	    priorExists: boolean;
	    suggestedRequest: number;
	
	    static createFrom(source: any = {}) {
	        return new ForecastRowDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vaccineId = source["vaccineId"];
	        this.name = source["name"];
	        this.code = source["code"];
	        this.unit = source["unit"];
	        this.initialized = source["initialized"];
	        this.method = source["method"];
	        this.sampleMonths = source["sampleMonths"];
	        this.sampleCount = source["sampleCount"];
	        this.estimatedDemand = source["estimatedDemand"];
	        this.openingSuggestion = source["openingSuggestion"];
	        this.priorMonth = source["priorMonth"];
	        this.priorExists = source["priorExists"];
	        this.suggestedRequest = source["suggestedRequest"];
	    }
	}
	export class ForecastDTO {
	    targetMonth: string;
	    generatedAt: string;
	    lowStockThreshold: number;
	    rows: ForecastRowDTO[];
	
	    static createFrom(source: any = {}) {
	        return new ForecastDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetMonth = source["targetMonth"];
	        this.generatedAt = source["generatedAt"];
	        this.lowStockThreshold = source["lowStockThreshold"];
	        this.rows = this.convertValues(source["rows"], ForecastRowDTO);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class InitializeInventoryInputDTO {
	    vaccineId: number;
	    month: string;
	    openingBalance: number;
	    stockReceived: number;
	    overrideReason: string;
	
	    static createFrom(source: any = {}) {
	        return new InitializeInventoryInputDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vaccineId = source["vaccineId"];
	        this.month = source["month"];
	        this.openingBalance = source["openingBalance"];
	        this.stockReceived = source["stockReceived"];
	        this.overrideReason = source["overrideReason"];
	    }
	}
	export class MonthTotalsDTO {
	    openingBalance: number;
	    stockReceived: number;
	    usageTotal: number;
	    remaining: number;
	    activeVaccines: number;
	    initializedVaccines: number;
	    uninitializedActive: number;
	    lowStockCount: number;
	
	    static createFrom(source: any = {}) {
	        return new MonthTotalsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.openingBalance = source["openingBalance"];
	        this.stockReceived = source["stockReceived"];
	        this.usageTotal = source["usageTotal"];
	        this.remaining = source["remaining"];
	        this.activeVaccines = source["activeVaccines"];
	        this.initializedVaccines = source["initializedVaccines"];
	        this.uninitializedActive = source["uninitializedActive"];
	        this.lowStockCount = source["lowStockCount"];
	    }
	}
	export class MonthRowDTO {
	    vaccineId: number;
	    vaccineName: string;
	    vaccineCode: string;
	    unit: string;
	    archived: boolean;
	    initialized: boolean;
	    status: string;
	    openingBalance: number;
	    stockReceived: number;
	    usageTotal: number;
	    remaining: number;
	    closingBalance: number;
	    hasClosingBalance: boolean;
	    carryoverOverridden: boolean;
	    carryoverReason: string;
	    suggestedOpening: number;
	    priorMonth: string;
	    priorExists: boolean;
	    priorClosed: boolean;
	    dailyUsage: number[];
	    lowStock: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MonthRowDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vaccineId = source["vaccineId"];
	        this.vaccineName = source["vaccineName"];
	        this.vaccineCode = source["vaccineCode"];
	        this.unit = source["unit"];
	        this.archived = source["archived"];
	        this.initialized = source["initialized"];
	        this.status = source["status"];
	        this.openingBalance = source["openingBalance"];
	        this.stockReceived = source["stockReceived"];
	        this.usageTotal = source["usageTotal"];
	        this.remaining = source["remaining"];
	        this.closingBalance = source["closingBalance"];
	        this.hasClosingBalance = source["hasClosingBalance"];
	        this.carryoverOverridden = source["carryoverOverridden"];
	        this.carryoverReason = source["carryoverReason"];
	        this.suggestedOpening = source["suggestedOpening"];
	        this.priorMonth = source["priorMonth"];
	        this.priorExists = source["priorExists"];
	        this.priorClosed = source["priorClosed"];
	        this.dailyUsage = source["dailyUsage"];
	        this.lowStock = source["lowStock"];
	    }
	}
	export class MonthOverviewDTO {
	    month: string;
	    days: number;
	    state: string;
	    rows: MonthRowDTO[];
	    totals: MonthTotalsDTO;
	    lowStockThreshold: number;
	    adjustments: AdjustmentDTO[];
	    generatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new MonthOverviewDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.days = source["days"];
	        this.state = source["state"];
	        this.rows = this.convertValues(source["rows"], MonthRowDTO);
	        this.totals = this.convertValues(source["totals"], MonthTotalsDTO);
	        this.lowStockThreshold = source["lowStockThreshold"];
	        this.adjustments = this.convertValues(source["adjustments"], AdjustmentDTO);
	        this.generatedAt = source["generatedAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class VaccineStatDTO {
	    vaccineId: number;
	    name: string;
	    code: string;
	    unit: string;
	    archived: boolean;
	    initialized: boolean;
	    opening: number;
	    received: number;
	    usageTotal: number;
	    remaining: number;
	    previousUsage: number;
	    usageChange: number;
	    lowStock: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VaccineStatDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vaccineId = source["vaccineId"];
	        this.name = source["name"];
	        this.code = source["code"];
	        this.unit = source["unit"];
	        this.archived = source["archived"];
	        this.initialized = source["initialized"];
	        this.opening = source["opening"];
	        this.received = source["received"];
	        this.usageTotal = source["usageTotal"];
	        this.remaining = source["remaining"];
	        this.previousUsage = source["previousUsage"];
	        this.usageChange = source["usageChange"];
	        this.lowStock = source["lowStock"];
	    }
	}
	export class MonthReportDTO {
	    month: string;
	    state: string;
	    totals: MonthTotalsDTO;
	    perVaccine: VaccineStatDTO[];
	    daily: DailyTotalDTO[];
	    previousMonth: string;
	    previousUsage: number;
	    usageChange: number;
	    hasComparison: boolean;
	    changePercent: number;
	    changePercentAvailable: boolean;
	    adjustmentCount: number;
	    generatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new MonthReportDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.state = source["state"];
	        this.totals = this.convertValues(source["totals"], MonthTotalsDTO);
	        this.perVaccine = this.convertValues(source["perVaccine"], VaccineStatDTO);
	        this.daily = this.convertValues(source["daily"], DailyTotalDTO);
	        this.previousMonth = source["previousMonth"];
	        this.previousUsage = source["previousUsage"];
	        this.usageChange = source["usageChange"];
	        this.hasComparison = source["hasComparison"];
	        this.changePercent = source["changePercent"];
	        this.changePercentAvailable = source["changePercentAvailable"];
	        this.adjustmentCount = source["adjustmentCount"];
	        this.generatedAt = source["generatedAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class PatientDTO {
	    id: number;
	    fullName: string;
	    patientCode: string;
	    archived: boolean;
	    archivedAt: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new PatientDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fullName = source["fullName"];
	        this.patientCode = source["patientCode"];
	        this.archived = source["archived"];
	        this.archivedAt = source["archivedAt"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class PatientInputDTO {
	    fullName: string;
	    patientCode: string;
	
	    static createFrom(source: any = {}) {
	        return new PatientInputDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fullName = source["fullName"];
	        this.patientCode = source["patientCode"];
	    }
	}
	export class PublicationPreviewDTO {
	    month: string;
	    monthState: string;
	    snapshot: publishing.Snapshot;
	    warnings: string[];
	    canPublish: boolean;
	    includedRows: number;
	    generatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicationPreviewDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.monthState = source["monthState"];
	        this.snapshot = this.convertValues(source["snapshot"], publishing.Snapshot);
	        this.warnings = source["warnings"];
	        this.canPublish = source["canPublish"];
	        this.includedRows = source["includedRows"];
	        this.generatedAt = source["generatedAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PublicationRecordDTO {
	    id: number;
	    snapshotId: string;
	    month: string;
	    status: string;
	    endpoint: string;
	    version: number;
	    payloadSha256: string;
	    publishedAt: string;
	    unpublishedAt: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicationRecordDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.snapshotId = source["snapshotId"];
	        this.month = source["month"];
	        this.status = source["status"];
	        this.endpoint = source["endpoint"];
	        this.version = source["version"];
	        this.payloadSha256 = source["payloadSha256"];
	        this.publishedAt = source["publishedAt"];
	        this.unpublishedAt = source["unpublishedAt"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class PublicationResultDTO {
	    ok: boolean;
	    code: string;
	    field: string;
	    message: string;
	    snapshotId: string;
	    version: number;
	    publishedAt: string;
	    duplicate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PublicationResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.code = source["code"];
	        this.field = source["field"];
	        this.message = source["message"];
	        this.snapshotId = source["snapshotId"];
	        this.version = source["version"];
	        this.publishedAt = source["publishedAt"];
	        this.duplicate = source["duplicate"];
	    }
	}
	export class PublicationStatusDTO {
	    endpointConfigured: boolean;
	    publisherTokenConfigured: boolean;
	    publisherTokenStorage: string;
	    latest?: PublicationRecordDTO;
	    history: PublicationRecordDTO[];
	
	    static createFrom(source: any = {}) {
	        return new PublicationStatusDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.endpointConfigured = source["endpointConfigured"];
	        this.publisherTokenConfigured = source["publisherTokenConfigured"];
	        this.publisherTokenStorage = source["publisherTokenStorage"];
	        this.latest = this.convertValues(source["latest"], PublicationRecordDTO);
	        this.history = this.convertValues(source["history"], PublicationRecordDTO);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SettingsDTO {
	    lowStockThreshold: number;
	    publisherEndpoint: string;
	    publisherTokenConfigured: boolean;
	    publisherTokenStorage: string;
	    dataDir: string;
	    databasePath: string;
	    backupDir: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lowStockThreshold = source["lowStockThreshold"];
	        this.publisherEndpoint = source["publisherEndpoint"];
	        this.publisherTokenConfigured = source["publisherTokenConfigured"];
	        this.publisherTokenStorage = source["publisherTokenStorage"];
	        this.dataDir = source["dataDir"];
	        this.databasePath = source["databasePath"];
	        this.backupDir = source["backupDir"];
	    }
	}
	export class UpdateInventoryInputDTO {
	    vaccineId: number;
	    month: string;
	    openingBalance: number;
	    stockReceived: number;
	    overrideReason: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInventoryInputDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vaccineId = source["vaccineId"];
	        this.month = source["month"];
	        this.openingBalance = source["openingBalance"];
	        this.stockReceived = source["stockReceived"];
	        this.overrideReason = source["overrideReason"];
	    }
	}
	export class UpdateUsageInputDTO {
	    id: number;
	    patientId: number;
	    administeredOn: string;
	    quantity: number;
	    note: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateUsageInputDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.patientId = source["patientId"];
	        this.administeredOn = source["administeredOn"];
	        this.quantity = source["quantity"];
	        this.note = source["note"];
	        this.reason = source["reason"];
	    }
	}
	export class UsageEntryDTO {
	    id: number;
	    patientId: number;
	    vaccineId: number;
	    administeredOn: string;
	    month: string;
	    quantity: number;
	    note: string;
	    voided: boolean;
	    voidReason: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new UsageEntryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.patientId = source["patientId"];
	        this.vaccineId = source["vaccineId"];
	        this.administeredOn = source["administeredOn"];
	        this.month = source["month"];
	        this.quantity = source["quantity"];
	        this.note = source["note"];
	        this.voided = source["voided"];
	        this.voidReason = source["voidReason"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class UsageRowDTO {
	    id: number;
	    patientId: number;
	    vaccineId: number;
	    administeredOn: string;
	    month: string;
	    quantity: number;
	    note: string;
	    voided: boolean;
	    voidReason: string;
	    createdAt: string;
	    updatedAt: string;
	    patientName: string;
	    patientCode: string;
	    patientArchived: boolean;
	    vaccineName: string;
	    vaccineCode: string;
	    vaccineUnit: string;
	
	    static createFrom(source: any = {}) {
	        return new UsageRowDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.patientId = source["patientId"];
	        this.vaccineId = source["vaccineId"];
	        this.administeredOn = source["administeredOn"];
	        this.month = source["month"];
	        this.quantity = source["quantity"];
	        this.note = source["note"];
	        this.voided = source["voided"];
	        this.voidReason = source["voidReason"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.patientName = source["patientName"];
	        this.patientCode = source["patientCode"];
	        this.patientArchived = source["patientArchived"];
	        this.vaccineName = source["vaccineName"];
	        this.vaccineCode = source["vaccineCode"];
	        this.vaccineUnit = source["vaccineUnit"];
	    }
	}
	export class VaccineDTO {
	    id: number;
	    name: string;
	    code: string;
	    description: string;
	    category: string;
	    unit: string;
	    archived: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new VaccineDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.code = source["code"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.unit = source["unit"];
	        this.archived = source["archived"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class VaccineInputDTO {
	    name: string;
	    code: string;
	    description: string;
	    category: string;
	    unit: string;
	
	    static createFrom(source: any = {}) {
	        return new VaccineInputDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.code = source["code"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.unit = source["unit"];
	    }
	}
	
	export class VoidUsageInputDTO {
	    id: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new VoidUsageInputDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.reason = source["reason"];
	    }
	}

}

export namespace publishing {
	
	export class Vaccine {
	    name: string;
	    code?: string;
	    unit: string;
	    openingBalance: number;
	    stockReceived: number;
	    totalUsage: number;
	    remaining: number;
	
	    static createFrom(source: any = {}) {
	        return new Vaccine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.code = source["code"];
	        this.unit = source["unit"];
	        this.openingBalance = source["openingBalance"];
	        this.stockReceived = source["stockReceived"];
	        this.totalUsage = source["totalUsage"];
	        this.remaining = source["remaining"];
	    }
	}
	export class Totals {
	    vaccineCount: number;
	    openingBalance: number;
	    stockReceived: number;
	    totalUsage: number;
	    remaining: number;
	
	    static createFrom(source: any = {}) {
	        return new Totals(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vaccineCount = source["vaccineCount"];
	        this.openingBalance = source["openingBalance"];
	        this.stockReceived = source["stockReceived"];
	        this.totalUsage = source["totalUsage"];
	        this.remaining = source["remaining"];
	    }
	}
	export class Snapshot {
	    schemaVersion: number;
	    snapshotId: string;
	    month: string;
	    aggregation: string;
	    generatedAt: string;
	    totals: Totals;
	    vaccines: Vaccine[];
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.snapshotId = source["snapshotId"];
	        this.month = source["month"];
	        this.aggregation = source["aggregation"];
	        this.generatedAt = source["generatedAt"];
	        this.totals = this.convertValues(source["totals"], Totals);
	        this.vaccines = this.convertValues(source["vaccines"], Vaccine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

