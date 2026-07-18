export interface EvidenceSnapshot {
  id: number
  signalId: number
  sourceUrl: string
  evidenceClass: string
  title?: string
  excerpt?: string
  contentHash?: string
  capturedAt: string
  createdAt?: string
}

export interface SignalAnalysis {
  matchedTopics?: string[]
  valueScore?: number
  evidenceClass?: string
  facts?: Array<{ claim: string; sourceUrl: string }>
  whatChanged?: string
  audience?: string
  practicalUse?: string
  prerequisites?: string
  action?: string
  painPoint?: string
  contentOpportunity?: string
  toolType?: string
  compatibleClients?: string[]
  installation?: string
  uncertainty?: string
  alertEligible?: boolean
  alertCategory?: string
  alertReason?: string
  [key: string]: unknown
}

export interface RadarSignal {
  id: number
  source: string
  title: string
  originalUrl: string
  author?: string
  score: number
  rankScore: number
  qualification: string
  qualificationReason?: string
  lifecycleState: string
  sourcePublishedAt?: string
  sourceUpdatedAt?: string
  createdAt: string
  evidence?: EvidenceSnapshot
  analysis?: SignalAnalysis
}

export interface SourceConfig {
  source: string
  enabled: boolean
  settings: {
    communities?: string[]
    repositories?: string[]
    feeds?: string[]
  }
  lastSuccessAt?: string
  lastFailure?: string
  lastRun?: {
    status: string
    itemCount: number
    durationMs: number
    failureReason?: string
    startedAt: string
    finishedAt?: string
  }
  updatedAt: string
}

export interface ContentStrategy {
  angle: string
  audience: string
  evidenceNote: string
}

export interface PlatformDraft {
  title: string
  body: string
  tags: string[]
  sourceLinks: string[]
}

export interface XDraft {
  chinese: string
  english: string
  sourceLinks: string[]
}

export interface VisualAsset {
  purpose: string
  aspectRatio: string
  prompt: string
}

export interface ContentPackage {
  id: number
  signalId: number
  evidenceSnapshotId: number
  status: 'draft' | 'approved'
  strategy: ContentStrategy
  xiaohongshu: PlatformDraft
  wechat: PlatformDraft
  x: XDraft
  visualPlan: VisualAsset[]
  approvedAt?: string
  createdAt: string
  updatedAt: string
}
