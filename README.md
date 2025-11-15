# oci-func-delete-objectstorage-olditems

OCI Function to Delete Old Objects in Object Storage

This is a Go-based Oracle Cloud Infrastructure (OCI) Function that recursively deletes objects older than a specified retention period from an Object Storage bucket. It uses Resource Principal authentication and is designed to run as a scheduled function (e.g., via OCI Events or cron-like triggers) for automated cleanup of old logs or data.

## Repository

https://github.com/mattn/oci-func-delete-objectstorage-olditems

## Features

- **Recursive Deletion**: Scans the entire bucket (including subdirectories) and deletes objects older than the configured retention days.
- **Configurable**: Bucket name and retention period set via environment variables or JSON input payload.
- **Error Handling**: Logs errors for failed deletions and continues processing.
- **Efficient Listing**: Uses OCI SDK's `ListObjects` with delimiters for directory traversal.
- **Output Logging**: Prints deleted objects and a summary count to the function's output stream.

## Prerequisites

- **OCI Account**: Access to OCI Console with permissions to create Functions, Dynamic Groups, and Policies.
- **OCI CLI and Fn Project**: Installed locally for development and deployment (see [OCI Functions Quickstart](https://docs.oracle.com/en-us/iaas/Content/Functions/Concepts/funconcepts.htm)).
- **Go 1.21+**: For local building (optional, as Functions builds from source).
- **Dependencies**: The code uses `github.com/oracle/oci-go-sdk/v65` for OCI interactions and `github.com/fnproject/fdk-go` for the function handler.

To operate Object Storage from Oracle Cloud Functions, you must set up a **Dynamic Group** and **Policy** for Resource Principal authentication:

### Dynamic Group

1. In OCI Console, navigate to **Identity & Security > Dynamic Groups > Create Dynamic Group**.
2. Name: `oci-func-delete-objectstorage-olditems-dynamic-group`
3. Matching Rule: 
   ```
   resource.id = 'ocid1.fnfunc.oc1....'
   ```
   (Replace with your actual Function OCID, e.g., `ocid1.fnfunc.oc1.iad.anuwcljr6q7x...`. You can find this in the Functions app details after deployment.)
4. Create the group.

### Policy

1. In OCI Console, navigate to **Identity & Security > Policies > Create Policy**.
2. Name: e.g., `OCI-Function-ObjectStorage-Policy`
3. Compartment: Root or the tenancy compartment.
4. Policy Statement:
   ```
   Allow dynamic-group 'oci-func-delete-objectstorage-olditems-dynamic-group' to manage object-family in tenancy
   ```
   (This grants read/list/delete permissions on all objects in the tenancy. Narrow it to a specific compartment or bucket if needed, e.g., `in compartment <compartment-ocid>`.)
5. Create the policy. Propagation may take a few minutes.

Without these, the function will fail with authorization errors (e.g., 403 Forbidden).

## Installation and Deployment

1. **Clone the Repo**:
   ```
   git clone https://github.com/mattn/oci-func-delete-objectstorage-olditems
   cd oci-func-delete-objectstorage-olditems
   ```

2. **Set Up OCI CLI**:
   - Configure OCI CLI with `oci setup config`.
   - Install Fn Project: Follow [OCI Functions Setup](https://docs.oracle.com/en-us/iaas/Content/Functions/Tasks/functionsoci.htm).

3. **Create Functions App**:
   ```
   fn create app <app-name> --annotation oracle.com/oci/subnetIds='["ocid1.subnet.oc1.iad.aaaa..."]'
   ```
   (Replace with your VCN subnet OCID for private access if needed.)

4. **Configure Environment Variables** (in the app; optional):
   ```
   fn config app <app-name> BUCKET_NAME "your-bucket-name"
   fn config app <app-name> RETENTION_DAYS "30"
   ```
   - `BUCKET_NAME`: The OCI Object Storage bucket to clean (optional; defaults to JSON input or error if unspecified).
   - `RETENTION_DAYS`: Days to retain objects (optional; default: 30; integer).

   Environment variables are not required. Parameters can also be provided via JSON input payload (see Usage).

5. **Deploy the Function**:
   ```
   fn deploy --app <app-name>
   ```

6. **Test Invocation**:
   ```
   fn invoke <app-name> oci-func-delete-objectstorage-olditems
   ```
   Check logs in OCI Console > Developer Services > Functions > Application Logs.

## Usage

- **Triggering**: Invoke manually via Fn CLI, OCI Console, or schedule via OCI Events Service (e.g., every day at midnight).
- **Behavior**:
  - Lists objects recursively from bucket root.
  - Deletes those created before `now - RETENTION_DAYS`.
  - Outputs: "deleting: <object-name> (created: <time>)" for each deletion, and "Total deleted objects: <count> (bucket: <name>, retention: <days> days)".
- **Example Output**:
  ```
  deleting: logs/old-file.log (created: 2023-10-01T12:00:00Z)
  Total deleted objects: 5 (bucket: nostr-relay-logs, retention: 30 days)
  ```

- **Parameter Input Options**:
  - **Environment Variables**: Set `BUCKET_NAME` and `RETENTION_DAYS` as above (fallback values).
  - **JSON Input Payload**: Provide parameters directly in the function invocation body (overrides environment variables). Example:
    ```
    {"bucket-name": "my-bucket", "retention-days": 15}
    ```
    - `bucket-name`: String; the bucket to clean (required if no env var).
    - `retention-days`: Integer; days to retain (default: 30 if unspecified).

- **Scheduling with Resource Scheduler**: This function can be executed periodically using OCI Resource Scheduler. For dynamic parameter specification per invocation, provide the JSON input body in the schedule trigger:
  ```
  {"bucket-name": "my-bucket", "retention-days": 15}
  ```
  This allows overriding environment variables for each run, enabling flexible multi-bucket or custom-retention cleanup in a single function deployment.

## Limitations and Notes

- **Irreversible**: Deletions are permanent; test on a staging bucket first.
- **Large Buckets**: May timeout for very large buckets; consider pagination (`Limit` in ListObjects) for production.
- **Versioning**: If bucket versioning is enabled, deleted objects become previous versions—adjust policy if needed.
- **Costs**: Function invocations and Object Storage operations incur OCI charges.

## Troubleshooting

- **Auth Errors**: Verify Dynamic Group includes your Function's OCID and Policy is active.
- **ListObjects Fails**: Check bucket existence and permissions.
- **No Deletions**: Ensure objects exist and are older than retention period.
- **Logs**: Use `fn logs <app> -f` or OCI Console for detailed output.

## License

MIT

## Author

Yasuhiro Matsumoto (a.k.a. mattn)
