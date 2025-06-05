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
  RunLoadTestRequest,
  LoadTestSpec
} from "../gen/proto/ironbird_pb.js";

const transport = createGrpcWebTransport({
  baseUrl: import.meta.env.VITE_IRONBIRD_GRPC_ADDRESS || "http://localhost:9006",
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

  runLoadTest: async (workflowId: string, loadTestSpec: LoadTestSpec): Promise<WorkflowResponse> => {
    const request = new RunLoadTestRequest({
      workflowId: workflowId,
      loadTestSpec: loadTestSpec
    });
    return await client.runLoadTest(request) as WorkflowResponse;
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

  sendShutdownSignal: async (workflowId: string): Promise<WorkflowResponse> => {
    const request = new SignalWorkflowRequest({
      workflowId: workflowId,
      signalName: "shutdown"
    });
    return await client.signalWorkflow(request) as WorkflowResponse;
  }
};
