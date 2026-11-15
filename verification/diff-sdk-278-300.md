# Protocol surface diff 0.3.278 -> 0.3.300

## sdkMessages (12)
- **changed-shape** `assistant` [additive, stable]: +agent_id
- **changed-shape** `conversation_reset` [additive, stable]: +timestamp; +trigger; +user_message_uuid
- **changed-shape** `result:success` [additive, stable]: +first_text_post_queue_wait_ms; +first_text_post_queued_behind
- **changed-shape** `system:background_tasks_changed` [neutral, stable]: ~tasks
- **changed-shape** `system:informational` [additive, stable]: +tag
- **changed-shape** `system:init` [additive, stable]: +plugin_errors; +view_mode
- **changed-shape** `system:task_notification` [additive, stable]: +run_id
- **changed-shape** `system:task_progress` [additive, stable]: +run_id
- **changed-shape** `system:task_started` [additive, stable]: +parent_task_id; +run_id
- **changed-shape** `system:task_updated` [additive, stable]: +run_id
- **changed-shape** `user` [additive, stable]: +agent_id; +client_composed; +inline_pastes
- **changed-shape** `user#SDKUserMessageReplay` [additive, stable]: +client_composed

## stdoutWire (12)
- **changed-shape** `assistant` [additive, stable]: +agent_id
- **changed-shape** `conversation_reset` [additive, stable]: +timestamp; +trigger; +user_message_uuid
- **changed-shape** `result:success` [additive, stable]: +first_text_post_queue_wait_ms; +first_text_post_queued_behind
- **changed-shape** `system:background_tasks_changed` [neutral, stable]: ~tasks
- **changed-shape** `system:informational` [additive, stable]: +tag
- **changed-shape** `system:init` [additive, stable]: +plugin_errors; +view_mode
- **changed-shape** `system:task_notification` [additive, stable]: +run_id
- **changed-shape** `system:task_progress` [additive, stable]: +run_id
- **changed-shape** `system:task_started` [additive, stable]: +parent_task_id; +run_id
- **changed-shape** `system:task_updated` [additive, stable]: +run_id
- **changed-shape** `user` [additive, stable]: +agent_id; +client_composed; +inline_pastes
- **changed-shape** `user#SDKUserMessageReplay` [additive, stable]: +client_composed

## controlRequests (4)
- **changed-shape** `elicitation` [additive, stable]: +property_order
- **added** `get_task_output` [additive, stable] (SDKControlGetTaskOutputRequest: subtype, task_id)
- **changed-shape** `initialize` [additive, stable]: +sdkMcpServerManifestsOrigin
- **added** `mcp_read_resource` [additive, stable] (SDKControlMcpReadResourceRequest: serverName, subtype, uri)

## controlResponses (4)
- **added** `SDKControlGetTaskOutputResponse` [additive, stable] (SDKControlGetTaskOutputResponse: output, total_bytes, truncated)
- **changed-shape** `SDKControlGetUsageResponse` [neutral, stable]: ~behaviors
- **changed-shape** `SDKControlInitializeResponse` [additive, stable]: +claude_code_version; +sdk_mcp_manifests_parked
- **added** `SDKControlMcpReadResourceResponse` [additive, stable] (SDKControlMcpReadResourceResponse: contents)

## tools (14)
- **changed-shape** `AgentInput` [additive, stable]: +effort
- **changed-shape** `AgentOutput` [additive, stable]
- **changed-shape** `ArtifactInput` [neutral, stable]: ~action; +pin
- **changed-shape** `ArtifactOutput` [additive, stable]
- **changed-shape** `CronCreateOutput` [additive, stable]: +nextFireAt
- **changed-shape** `CronListOutput` [neutral, stable]: ~jobs
- **changed-shape** `FileReadInput` [additive, stable]: +allow_large
- **added** `OfferChromeSetupInput` [additive, stable] (None: reason)
- **added** `OfferChromeSetupOutput` [additive, stable] (None: outcome)
- **changed-shape** `ReadNotificationsOutput` [neutral, stable]: ~notifications; +read_at
- **changed-shape** `ToolInputSchemas` [additive, stable]
- **changed-shape** `ToolOutputSchemas` [additive, stable]
- **changed-shape** `WebFetchInput` [additive, stable]: +offset
- **changed-shape** `WebSearchOutput` [neutral, stable]: ~results

## options (1)
- **added** `verbatimPrompts` [additive, stable]: `boolean`

## queryMethods (1)
- **added** `readMcpResource` [additive, stable]: `(serverName: string, uri: string): Promise<SDKControlMcpReadResourceResponse>;`

## settings (9)
- **added** `allowClaudeInChromeWithManagedMcp` [additive, stable]: `boolean`
- **added** `allowedProviders` [additive, stable]: `('anthropic' | 'customEndpoint' | 'bedrock' | 'vertex' | 'foundry' | 'anthropicAws' | 'mantle' | 'gateway')[]`
- **changed-shape** `attribution` [neutral, stable]: type: `…` -> `boolean | …`
- **added** `availableModelsMatch` [additive, stable]: `'prefix' | 'exact'`
- **added** `deniedModels` [additive, stable]: `string[]`
- **changed-shape** `hooks` [neutral, stable]: type: `…bash' | 'powershell'; timeout?: number; statusMessage?: string; once?: boolean; async?: boolean; asyncRewake?: boolean; } | { type: 'prompt'; prompt: string; if…` -> `…bash' | 'powershell'; timeout?: number; onFailure?: 'continue' | 'block'; statusMessage?: string; once?: boolean; async?: boolean; asyncRewake?: boolean; } | { type: 'prompt'; prompt: string; if?: str…`
- **added** `idleCompaction` [additive, stable]: `boolean`
- **added** `maxProseWidth` [additive, stable]: `number`
- **changed-shape** `modelSettings` [neutral, stable]: type: `… | 'medium' | 'high' | 'xhigh' | 'max'; …` -> `… | 'medium' | 'high' | 'xhigh' | 'max'; autoCompactWindow?: 'auto' | number; …`

## exports (27)
- **changed-shape** `AgentDefinition` [neutral, stable]
- **added** `ClaimOptions` [additive, stable]
- **changed-shape** `McpServerStatus` [neutral, stable]
- **changed-shape** `NonNullableUsage` [neutral, stable]
- **changed-shape** `Options` [neutral, stable]
- **changed-shape** `Query` [neutral, stable]
- **changed-shape** `SDKAssistantMessage` [neutral, stable]
- **changed-shape** `SDKBackgroundTasksChangedMessage` [neutral, stable]
- **changed-shape** `SDKControlGetUsageResponse` [neutral, stable]
- **changed-shape** `SDKControlInitializeResponse` [neutral, stable]
- **added** `SDKControlMcpReadResourceResponse` [additive, stable]
- **changed-shape** `SDKConversationResetMessage` [neutral, stable]
- **changed-shape** `SDKInformationalMessage` [neutral, stable]
- **changed-shape** `SDKMessageOrigin` [neutral, stable]
- **changed-shape** `SDKRateLimitInfo` [neutral, stable]
- **changed-shape** `SDKResultSuccess` [neutral, stable]
- **changed-shape** `SDKStartupFailureReason` [neutral, stable]
- **changed-shape** `SDKSystemMessage` [neutral, stable]
- **changed-shape** `SDKTaskNotificationMessage` [neutral, stable]
- **changed-shape** `SDKTaskProgressMessage` [neutral, stable]
- **changed-shape** `SDKTaskStartedMessage` [neutral, stable]
- **changed-shape** `SDKTaskUpdatedMessage` [neutral, stable]
- **changed-shape** `SDKUserMessage` [neutral, stable]
- **changed-shape** `SDKUserMessageReplay` [neutral, stable]
- **changed-shape** `Settings` [neutral, stable]
- **added** `SpareProcess` [additive, stable]
- **added** `prewarm` [additive, stable]

## stringEnums (1)
- **changed-shape** `SDKStartupFailureReason` [additive, stable]: +['org_config_refused', 'org_config_required_unavailable', 'provider_not_allowed'] -[]

