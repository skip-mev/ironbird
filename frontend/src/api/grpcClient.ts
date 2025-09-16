import { createPromiseClient } from "@bufbuild/connect";
import { createGrpcWebTransport } from "@bufbuild/connect-web";
import { IronbirdService } from "../gen/proto/ironbird_connect.js";
import { 
  CreateWorkflowRequest, 
  WorkflowResponse, 
  GetWorkflowRequest, 
  Workflow, 
  ListWorkflowsRequest, 
  WorkflowListResponse,
  CancelWorkflowRequest,
  SignalWorkflowRequest,
  CreateWorkflowTemplateRequest,
  GetWorkflowTemplateRequest,
  ListWorkflowTemplatesRequest,
  UpdateWorkflowTemplateRequest,
  DeleteWorkflowTemplateRequest,
  WorkflowTemplateResponse,
  WorkflowTemplate,
  WorkflowTemplateListResponse,
  ExecuteWorkflowTemplateRequest,
  GetTemplateRunHistoryRequest,
  TemplateRunHistoryResponse,
} from "../gen/proto/ironbird_pb.js";

console.log("VITE_IRONBIRD_GRPC_ADDRESS:", import.meta.env.VITE_IRONBIRD_GRPC_ADDRESS);

const transport = createGrpcWebTransport({
  baseUrl: import.meta.env.VITE_IRONBIRD_GRPC_ADDRESS || "http://localhost:9007",
  credentials: "omit",
  interceptors: [
    (next) => async (req) => {
      console.log("gRPC request:", req.method, req.url);
      try {
        const response = await next(req);
        console.log("gRPC response:", response.message);
        return response;
      } catch (error) {
        console.error("gRPC error:", error);
        throw error;
      }
    }
  ]
});

// Create the client
export const client = createPromiseClient(IronbirdService, transport);

export const grpcWorkflowApi = {
  createWorkflow: async (request: CreateWorkflowRequest): Promise<WorkflowResponse> => {
    try {
      console.log("Calling createWorkflow with:", request);
      const response = await client.createWorkflow(request) as WorkflowResponse;
      console.log("createWorkflow response:", response);
      return response;
    } catch (error) {
      console.error("createWorkflow error:", error);
      throw error;
    }
  },

  listWorkflows: async (limit: number = 100, offset: number = 0): Promise<WorkflowListResponse> => {
    try {
      console.log("Calling listWorkflows with limit:", limit, "offset:", offset);
      const request = new ListWorkflowsRequest({
        limit: limit,
        offset: offset
      });
      const response = await client.listWorkflows(request) as WorkflowListResponse;
      console.log("listWorkflows response:", response);
      return response;
    } catch (error) {
      console.error("listWorkflows error:", error);
      throw error;
    }
  },

  getWorkflow: async (workflowId: string): Promise<Workflow> => {
    const request = new GetWorkflowRequest({
      workflowId: workflowId
    });
    return await client.getWorkflow(request) as Workflow;
  },

  cancelWorkflow: async (workflowId: string): Promise<WorkflowResponse> => {
    const request = new CancelWorkflowRequest({
      workflowId: workflowId
    });
    return await client.cancelWorkflow(request) as WorkflowResponse;
  },

  signalWorkflow: async (workflowId: string, signalName: string): Promise<WorkflowResponse> => {
    const request = new SignalWorkflowRequest({
      workflowId: workflowId,
      signalName: signalName
    });
    return await client.signalWorkflow(request) as WorkflowResponse;
  },

  // Template management methods
  createWorkflowTemplate: async (request: CreateWorkflowTemplateRequest): Promise<WorkflowTemplateResponse> => {
    try {
      console.log("Calling createWorkflowTemplate with:", request);
      const response = await client.createWorkflowTemplate(request) as WorkflowTemplateResponse;
      console.log("createWorkflowTemplate response:", response);
      return response;
    } catch (error) {
      console.error("createWorkflowTemplate error:", error);
      throw error;
    }
  },

  getWorkflowTemplate: async (templateId: string): Promise<WorkflowTemplate> => {
    const request = new GetWorkflowTemplateRequest({
      id: templateId
    });
    return await client.getWorkflowTemplate(request) as WorkflowTemplate;
  },

  listWorkflowTemplates: async (limit?: number, offset?: number): Promise<WorkflowTemplateListResponse> => {
    const request = new ListWorkflowTemplatesRequest({
      limit: limit || 50,
      offset: offset || 0,
    });
    return await client.listWorkflowTemplates(request) as WorkflowTemplateListResponse;
  },

  updateWorkflowTemplate: async (request: UpdateWorkflowTemplateRequest): Promise<WorkflowTemplateResponse> => {
    try {
      console.log("Calling updateWorkflowTemplate with:", request);
      const response = await client.updateWorkflowTemplate(request) as WorkflowTemplateResponse;
      console.log("updateWorkflowTemplate response:", response);
      return response;
    } catch (error) {
      console.error("updateWorkflowTemplate error:", error);
      throw error;
    }
  },

  deleteWorkflowTemplate: async (templateId: string): Promise<WorkflowTemplateResponse> => {
    const request = new DeleteWorkflowTemplateRequest({
      id: templateId
    });
    return await client.deleteWorkflowTemplate(request) as WorkflowTemplateResponse;
  },

  executeWorkflowTemplate: async (request: ExecuteWorkflowTemplateRequest): Promise<WorkflowResponse> => {
    try {
      console.log("Calling executeWorkflowTemplate with:", request);
      const response = await client.executeWorkflowTemplate(request) as WorkflowResponse;
      console.log("executeWorkflowTemplate response:", response);
      return response;
    } catch (error) {
      console.error("executeWorkflowTemplate error:", error);
      throw error;
    }
  },

  getTemplateRunHistory: async (templateId: string, limit?: number, offset?: number): Promise<TemplateRunHistoryResponse> => {
    const request = new GetTemplateRunHistoryRequest({
      id: templateId,
      limit: limit || 50,
      offset: offset || 0
    });
    return await client.getTemplateRunHistory(request) as TemplateRunHistoryResponse;
  },
};
