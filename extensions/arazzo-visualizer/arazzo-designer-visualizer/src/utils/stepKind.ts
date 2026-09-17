/**
 * Copyright (c) 2025, WSO2 LLC. (https://www.wso2.com) All Rights Reserved.
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import { SourceDescription } from '@wso2/arazzo-designer-core';
import openApiIcon from '../resources/icons/openapi.svg';
import asyncApiIcon from '../resources/icons/asyncapi.svg';
import arazzoIcon from '../resources/icons/arazzo.svg';

export type StepKind = 'OpenAPI' | 'AsyncAPI' | 'Arazzo' | 'Workflow';

/** The logo drawn on a step node for each kind. A nested workflow is an Arazzo workflow too. */
export const STEP_KIND_ICONS: Record<StepKind, string> = {
    OpenAPI: openApiIcon,
    AsyncAPI: asyncApiIcon,
    Arazzo: arazzoIcon,
    Workflow: arazzoIcon,
};

const mapSourceType = (t?: string): StepKind | undefined =>
    t === 'asyncapi' ? 'AsyncAPI' : t === 'openapi' ? 'OpenAPI' : t === 'arazzo' ? 'Arazzo' : undefined;

// Extract the source-description NAME an `operationPath` points at. Its shape differs from the
// scoped `operationId` form, so this cannot reuse that branch's parsing:
//   catalog#/paths/~1products/get                          -> catalog   (bare name before '#')
//   '{$sourceDescriptions.catalog.url}#/paths/~1p/get'     -> catalog   (expression before '#')
//   $sourceDescriptions.asyncOrderApi.placeOrder           -> asyncOrderApi (no '#'; the form the
//                                                             spec's own AsyncAPI example uses)
// In the '#' forms the trailing segment is a FIELD of the source (".url") and is dropped; in the
// scoped-operationId form the trailing segment is the operation id — which is why that branch
// keeps it and this one does not.
const sourceNameFromOperationPath = (raw: string): string => {
    let ref = String(raw).trim().replace(/^['"]|['"]$/g, '').trim();
    const hash = ref.indexOf('#');
    if (hash >= 0) {
        ref = ref.slice(0, hash);
    }
    ref = ref.trim().replace(/^\{/, '').replace(/\}$/, '').trim();
    const prefix = '$sourceDescriptions.';
    return ref.startsWith(prefix) ? ref.slice(prefix.length).split('.')[0] : ref;
};

/**
 * What a step targets. Shared by the properties panel's *Step Type* and the step node's icon, so the
 * two cannot disagree.
 *
 * The language server RESOLVES this for us (`stepType` on the model it returns) by looking the target
 * up inside the declared specs, so prefer it whenever it is present. The fallback below reads only
 * the Arazzo text and therefore has to guess: a bare `operationId` names no source description, so
 * with more than one declared source it cannot know which one owns the operation. Keep it for the
 * cases the server leaves unresolved — a remote source, a file not yet indexed, an operation that
 * exists nowhere.
 *
 * Fallback classification:
 *  - channelPath OR action -> AsyncAPI (action only applies to async steps)
 *  - workflowId            -> Workflow (nested)
 *  - operationId scoped "$sourceDescriptions.<name>.*" -> that source's declared type
 *  - operationId (bare): if the document declares exactly one typed source, use it; else OpenAPI
 *  - operationPath         -> the referenced source's declared type (NOT always OpenAPI: the spec
 *                            words operationPath as "an operation" and its own AsyncAPI example
 *                            uses operationPath against a `type: asyncapi` source)
 */
export function getStepKind(step: any, sourceDescriptions: SourceDescription[] = []): StepKind | undefined {
    const sourceTypeByName = (name: string): StepKind | undefined =>
        mapSourceType(sourceDescriptions.find(sd => sd.name === name)?.type);

    // What the server resolved, when it could resolve anything.
    const resolved = step.stepType === 'workflow' ? 'Workflow' : mapSourceType(step.stepType);
    if (resolved) {
        return resolved;
    }

    if (step.channelPath || step.action) {
        return 'AsyncAPI';
    }
    if (step.workflowId) {
        return 'Workflow';
    }
    if (step.operationId) {
        const opId = String(step.operationId);
        if (opId.startsWith('$sourceDescriptions.')) {
            const name = opId.slice('$sourceDescriptions.'.length).split('.')[0];
            return sourceTypeByName(name) ?? 'OpenAPI';
        }
        // A bare operationId names no source, so it can only be attributed when the document
        // declares exactly ONE source. Filtering to typed sources first would mis-attribute
        // the step whenever a second, untyped source could equally own the operation.
        return (sourceDescriptions.length === 1 ? mapSourceType(sourceDescriptions[0].type) : undefined) ?? 'OpenAPI';
    }
    if (step.operationPath) {
        return sourceTypeByName(sourceNameFromOperationPath(String(step.operationPath))) ?? 'OpenAPI';
    }
    return undefined;
}
