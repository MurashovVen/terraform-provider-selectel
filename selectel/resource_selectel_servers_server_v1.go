package selectel

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/terraform-providers/terraform-provider-selectel/selectel/internal/api/servers"
	waiters "github.com/terraform-providers/terraform-provider-selectel/selectel/waiters/servers"
)

func resourceServersServerV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServersServerV1Create,
		ReadContext:   resourceServersServerV1Read,
		UpdateContext: resourceServersServerV1Update,
		DeleteContext: resourceServersServerV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceServersServerV1ImportState,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: resourceServersServerV1Schema(),
	}
}

func resourceServersServerV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dsClient, diagErr := getServersClient(d, meta)
	if diagErr != nil {
		return diagErr
	}

	partitionsConfigFromSchema, err := resourceServersServerV1ReadPartitionsConfig(d)
	if err != nil {
		return diag.FromErr(fmt.Errorf(
			"failed to read partitions config: %w", err,
		))
	}

	var (
		locationID      = d.Get(serversServerSchemaKeyLocationID).(string)
		osID            = d.Get(serversServerSchemaKeyOSID).(string)
		configurationID = d.Get(serversServerSchemaKeyConfigurationID).(string)
		pricePlanName   = d.Get(serversServerSchemaKeyPricePlanName).(string)
		sshKeyName, _   = d.Get(serversServerSchemaKeyOSSSHKeyName).(string)

		isServerChip, _   = d.Get(serversServerSchemaKeyIsServerChip).(bool)
		publicSubnetID, _ = d.Get(serversServerSchemaKeyPublicSubnetID).(string)
		privateSubnet, _  = d.Get(serversServerSchemaKeyPrivateSubnet).(string)
	)

	data, diagErr := resourceServersServerV1CreateLoadData(
		ctx, dsClient, locationID, osID, configurationID, publicSubnetID, privateSubnet,
		sshKeyName, pricePlanName, isServerChip, partitionsConfigFromSchema,
	)
	if diagErr != nil {
		return diagErr
	}

	// validating availability of the server, OS, price plan and balance, partitions config

	var (
		userScript, _ = d.Get(serversServerSchemaKeyOSUserScript).(string)
		sshKeyPK, _   = d.Get(serversServerSchemaKeyOSSSHKey).(string)
	)

	if data.sshKeyByName != nil {
		sshKeyPK = data.sshKeyByName.PublicKey
	}

	diagErr = resourceServersServerV1CreateValidatePreconditions(
		ctx, dsClient, data, locationID, data.pricePlan.UUID, configurationID, osID, userScript != "",
		sshKeyPK != "" || data.sshKeyByName != nil, isServerChip, privateSubnet != "",
	)
	if diagErr != nil {
		return diagErr
	}

	// creating

	serverObjectName := objectServer
	if isServerChip {
		serverObjectName = objectServerChip
	}

	var (
		hostName = resourceServersServerV1GenerateHostNameIfNotPresented(d)

		password, _ = d.Get(serversServerSchemaKeyOSPassword).(string)

		req = &servers.ServerBillingPostPayload{
			ServiceUUID:      configurationID,
			PricePlanUUID:    data.pricePlan.UUID,
			PayCurrency:      data.billingPayCurrency,
			LocationUUID:     locationID,
			Quantity:         1,
			IPList:           data.ipsPublic,
			LocalIPList:      data.ipsPrivate,
			LocalSubnetUUID:  data.localSubnetUUID,
			ProjectUUID:      d.Get(serversServerSchemaKeyProjectID).(string),
			PartitionsConfig: data.partitions,
			OSVersion:        data.os.VersionValue,
			OSTemplate:       data.os.OSValue,
			OSArch:           data.os.Arch,
			UserSSHKey:       sshKeyPK,
			UserHostname:     hostName,
			UserDesc:         hostName,
			Password:         password,
			UserScript:       userScript,
		}
	)

	log.Print(msgCreate(serverObjectName, req))

	billingRes, _, err := dsClient.ServerBilling(ctx, req, isServerChip)
	if err != nil {
		return diag.FromErr(fmt.Errorf(
			"failed to create %s %s: %w", serverObjectName, configurationID, err,
		))
	}

	switch {
	case len(billingRes) > 1:
		return diag.FromErr(fmt.Errorf(
			"failed to create one %s %s: multiple resources created: %#v", serverObjectName, configurationID, billingRes,
		))

	case len(billingRes) == 0:
		return diag.FromErr(fmt.Errorf(
			"failed to create %s %s: no resource returned", serverObjectName, configurationID,
		))
	}

	uuid := billingRes[0].UUID

	d.SetId(uuid)

	log.Printf("[DEBUG] waiting for server %s to become 'ACTIVE'", uuid)

	timeout := d.Timeout(schema.TimeoutCreate)
	err = waiters.WaitForServersServerV1ActiveState(ctx, dsClient, uuid, timeout)
	if err != nil {
		return diag.FromErr(errCreatingObject(serverObjectName, err))
	}

	return nil
}

type serversServerV1CreateData struct {
	os                 *servers.OperatingSystem
	server             *servers.Server
	partitions         servers.PartitionsConfig
	ipsPublic          []net.IP
	ipsPrivate         []net.IP
	localSubnetUUID    string
	sshKeyByName       *servers.SSHKey
	billing            *servers.ServiceBilling
	billingPayCurrency string
	pricePlan          *servers.PricePlan
}

func resourceServersServerV1CreateLoadData(
	ctx context.Context, dsClient *servers.ServiceClient,
	locationID, osID, configurationID, publicSubnetID, privateSubnet, sshKeyName, pricePlanName string, isServerChip bool,
	partitionsConfigFromSchema *PartitionsConfig,
) (*serversServerV1CreateData, diag.Diagnostics) {
	operatingSystems, _, err := dsClient.OperatingSystems(ctx, servers.OperatingSystemsQuery{
		LocationID: locationID,
		ServiceID:  configurationID,
	})
	if err != nil {
		return nil, diag.FromErr(errGettingObject(objectOS, osID, err))
	}

	os := operatingSystems.FindOneByID(osID)
	if os == nil {
		return nil, diag.FromErr(errGettingObject(objectOS, osID, ErrNotFound))
	}

	objectServerName := objectServer
	if isServerChip {
		objectServerName = objectServerChip
	}

	server, _, err := dsClient.ServerByID(ctx, configurationID, isServerChip)
	if err != nil {
		return nil, diag.FromErr(errGettingObject(objectServerName, configurationID, err))
	}

	var partitionsConfig servers.PartitionsConfig
	if !partitionsConfigFromSchema.IsEmpty() || os.Partitioning {
		if !os.Partitioning { // in case of configured partitions
			return nil, diag.FromErr(fmt.Errorf(
				"%s %s does not support partitions config", objectOS, os.OSValue,
			))
		}

		localDrives, _, err := dsClient.LocalDrives(ctx, configurationID)
		if err != nil {
			return nil, diag.FromErr(fmt.Errorf(
				"failed to get local drives for %s %s: %w", objectServerName, configurationID, err,
			))
		}

		partitionsConfig, err = partitionsConfigFromSchema.CastToAPIPartitionsConfig(localDrives, os.DefaultPartitions)
		if err != nil {
			return nil, diag.FromErr(fmt.Errorf(
				"failed to read partitions config input: %w", err,
			))
		}
	}

	var publicIPs []net.IP
	if publicSubnetID != "" {
		// also validating the sufficiency of free addresses
		publicIP, diagErr := resourceServersServerV1GetFreePublicIPs(ctx, dsClient, locationID, publicSubnetID)
		if diagErr != nil {
			return nil, diagErr
		}

		publicIPs = append(publicIPs, publicIP)
	}

	var (
		privateIPs      []net.IP
		localSubnetUUID string
	)
	if privateSubnet != "" {
		// also validating the sufficiency of free addresses
		var (
			diagErr   diag.Diagnostics
			privateIP net.IP
		)
		privateIP, localSubnetUUID, diagErr = resourceServersServerV1GetFreePrivateIPs(ctx, dsClient, locationID, privateSubnet)
		if diagErr != nil {
			return nil, diagErr
		}

		privateIPs = append(privateIPs, privateIP)
	}

	var sshKey *servers.SSHKey
	if sshKeyName != "" {
		sshKeys, _, err := dsClient.SSHKeys(ctx)
		if err != nil {
			return nil, diag.FromErr(fmt.Errorf(
				"failed to get SSH keys: %w", err,
			))
		}

		sshKey = sshKeys.FindOneByName(sshKeyName)
		if sshKey == nil {
			return nil, diag.FromErr(fmt.Errorf(
				"SSH key %s not found", sshKeyName,
			))
		}
	}

	pricePlans, _, err := dsClient.PricePlans(ctx)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf(
			"failed to get price plans: %w", err,
		))
	}

	pricePlan := pricePlans.FindOneByName(pricePlanName)
	if pricePlan == nil {
		return nil, diag.FromErr(fmt.Errorf(
			"price plan %s not found", pricePlanName,
		))
	}

	billing, _, err := dsClient.ServerCalculateBilling(ctx, configurationID, locationID, pricePlan.UUID, servers.ServiceBillingPayCurrencyMain, isServerChip)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf(
			"can't calculate billing for %s %s: %w", objectServerName, configurationID, err,
		))
	}

	billingPayCurrency := servers.ServiceBillingPayCurrencyMain

	if !billing.HasEnoughBalance {
		billing, _, err = dsClient.ServerCalculateBilling(ctx, configurationID, locationID, pricePlan.UUID, servers.ServiceBillingPayCurrencyBonus, isServerChip)
		if err != nil {
			return nil, diag.FromErr(fmt.Errorf(
				"can't calculate billing for %s %s: %w", objectServerName, configurationID, err,
			))
		}

		billingPayCurrency = servers.ServiceBillingPayCurrencyBonus
	}

	return &serversServerV1CreateData{
		os:                 os,
		server:             server,
		partitions:         partitionsConfig,
		ipsPublic:          publicIPs,
		ipsPrivate:         privateIPs,
		localSubnetUUID:    localSubnetUUID,
		sshKeyByName:       sshKey,
		billing:            billing,
		billingPayCurrency: billingPayCurrency,
		pricePlan:          pricePlan,
	}, nil
}

func resourceServersServerV1CreateValidatePreconditions(
	ctx context.Context, dsClient *servers.ServiceClient,
	data *serversServerV1CreateData,
	locationID, pricePlanID, configurationID, osID string,
	needUserScript, sshKey, isServerChip bool,
	needPrivateIP bool,
) diag.Diagnostics {
	objectServerName := objectServer
	if isServerChip {
		objectServerName = objectServerChip
	}

	switch {
	case !data.server.IsLocationAvailable(locationID):
		return diag.FromErr(fmt.Errorf(
			"%s %s is not available for %s %s", objectLocation, locationID, objectServerName, configurationID,
		))

	case !data.server.IsPricePlanAvailableForLocation(pricePlanID, locationID):
		return diag.FromErr(fmt.Errorf(
			"price-plan %s is not available for %s %s in %s %s",
			pricePlanID, objectServerName, configurationID, objectLocation, locationID,
		))

	case data.os == nil:
		return diag.FromErr(fmt.Errorf(
			"%s %s is not available for %s %s in %s %s",
			objectOS, osID, objectServerName, configurationID, objectLocation, locationID,
		))

	case needUserScript && !data.os.ScriptAllowed:
		return diag.FromErr(fmt.Errorf(
			"%s %s does not allow scripts", objectOS, osID,
		))

	case sshKey && !data.os.IsSSHKeyAllowed:
		return diag.FromErr(fmt.Errorf(
			"%s %s does not allow SSH keys", objectOS, osID,
		))

	case data.partitions != nil && !data.os.Partitioning:
		return diag.FromErr(fmt.Errorf(
			"%s %s does not support partitions config", objectOS, data.os.OSValue,
		))

	case !data.billing.HasEnoughBalance:
		return diag.FromErr(fmt.Errorf(
			"%s %s is not available for price-plan %s in %s %s because of insufficient balance (main, bonus)",
			objectServerName, configurationID, pricePlanID, objectLocation, locationID,
		))

	case needPrivateIP && !data.server.IsPrivateNetworkAvailable():
		return diag.FromErr(fmt.Errorf(
			"%s %s does not support private network", objectServerName, configurationID,
		))

	case needPrivateIP && !data.os.IsPrivateNetworkAvailable():
		return diag.FromErr(fmt.Errorf(
			"%s %s does not support private network", objectOS, osID,
		))
	}

	_, _, err := dsClient.PartitionsValidate(ctx, data.partitions, configurationID)
	if err != nil {
		return diag.FromErr(fmt.Errorf(
			"failed to validate partitions config for %s %s: %w", objectServerName, configurationID, err,
		))
	}

	return nil
}

func resourceServersServerV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dsClient, diagErr := getServersClient(d, meta)
	if diagErr != nil {
		return diagErr
	}

	log.Print(msgGet(objectServer, d.Id()))

	rd, _, err := dsClient.ResourceDetails(ctx, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf(
			"failed to read: %w", err,
		))
	}

	_ = d.Set("location_id", rd.LocationUUID)
	_ = d.Set("configuration_id", rd.ServiceUUID)
	_ = d.Set("price_plan_name", rd.Billing.CurrentPricePlan.Name)

	isServerChip := rd.IsServerChip()
	isServer := rd.IsServer()
	if !isServer && !isServerChip {
		return diag.FromErr(errors.New(
			"the resource is neither a server nor a server chip",
		))
	}

	_ = d.Set("is_server_chip", isServerChip)

	resourceOS, _, err := dsClient.OperatingSystemByResource(ctx, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf(
			"failed to read OS for server %s: %w", d.Id(), err,
		))
	}

	_ = d.Set("os_host_name", resourceOS.UserHostName)
	_ = d.Set("user_script", resourceOS.UserScript)
	_ = d.Set("os_password", resourceOS.Password)

	operatingSystems, _, err := dsClient.OperatingSystems(ctx, servers.OperatingSystemsQuery{
		LocationID: rd.LocationUUID,
		ServiceID:  rd.ServiceUUID,
	})
	if err != nil {
		return diag.FromErr(errGettingObjects(objectOS, err))
	}

	os := operatingSystems.FindOneByArchAndVersionAndOs(resourceOS.Arch, resourceOS.Version, resourceOS.OSValue)
	if os == nil {
		return diag.FromErr(
			fmt.Errorf("failed to find OS %s with arch %s and version %s", resourceOS.OSValue, resourceOS.Arch, resourceOS.Version),
		)
	}

	_ = d.Set("os_id", os.UUID)

	// todo think about ssh key

	return nil
}

func resourceServersServerV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dsClient, diagErr := getServersClient(d, meta)
	if diagErr != nil {
		return diagErr
	}

	log.Print(msgDelete(objectServer, d.Id()))

	_, err := dsClient.DeleteResource(ctx, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf(
			"failed to delete %s %s: %w", objectServer, d.Id(), err,
		))
	}

	log.Printf("[DEBUG] waiting for server %s to become 'EXPIRING'", d.Id())

	timeout := d.Timeout(schema.TimeoutCreate)
	err = waiters.WaitForServersServerV1RefusedToRenewState(ctx, dsClient, d.Id(), timeout)
	if err != nil {
		return diag.FromErr(errCreatingObject(objectServer, err))
	}

	return nil
}

func resourceServersServerV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dsClient, diagErr := getServersClient(d, meta)
	if diagErr != nil {
		return diagErr
	}

	var (
		locationID      = d.Get(serversServerSchemaKeyLocationID).(string)
		configurationID = d.Get(serversServerSchemaKeyConfigurationID).(string)
		osID            = d.Get(serversServerSchemaKeyOSID).(string)
		sshKeyName, _   = d.Get(serversServerSchemaKeyOSSSHKeyName).(string)
	)

	data, diagErr := resourceServersServerV1UpdateLoadData(ctx, dsClient, d, locationID, osID, configurationID, sshKeyName)
	if diagErr != nil {
		return diagErr
	}

	var (
		userScript, _ = d.Get(serversServerSchemaKeyOSUserScript).(string)
		sshKeyPK, _   = d.Get(serversServerSchemaKeyOSSSHKey).(string)
	)

	if data.sshKeyByName != nil {
		sshKeyPK = data.sshKeyByName.PublicKey
	}

	diagErr = resourceServersServerV1UpdateValidatePreconditions(
		ctx, d, dsClient, data.os, data.partitions, userScript != "", sshKeyPK != "" || data.sshKeyByName != nil,
	)
	if diagErr != nil {
		return diagErr
	}

	var (
		hostName = resourceServersServerV1GenerateHostNameIfNotPresented(d)

		password, _ = d.Get(serversServerSchemaKeyOSPassword).(string)

		payload = &servers.InstallNewOSPayload{
			OSVersion:        data.os.VersionValue,
			OSTemplate:       data.os.OSValue,
			OSArch:           data.os.Arch,
			UserSSHKey:       sshKeyPK,
			UserHostname:     hostName,
			Password:         password,
			PartitionsConfig: data.partitions,
			UserScript:       userScript,
		}
	)

	log.Print(msgUpdate(objectServer, d.Id(), payload))

	_, err := dsClient.InstallNewOS(ctx, payload, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf(
			"failed to update %s %s: %w", objectServer, d.Id(), err,
		))
	}

	log.Printf("[DEBUG] waiting for server %s to become 'ACTIVE'", d.Id())

	timeout := d.Timeout(schema.TimeoutCreate)
	err = waiters.WaitForServersServerInstallNewOSV1ActiveState(ctx, dsClient, d.Id(), timeout)
	if err != nil {
		return diag.FromErr(errUpdatingObject(objectServer, d.Id(), err))
	}

	// todo think about
	// imported state (no partitions) -> update some data -> default partitions
	// maybe need to get partitions from server before update
	// same case for ssh key

	return nil
}

type serversServerV1UpdateData struct {
	os           *servers.OperatingSystem
	partitions   servers.PartitionsConfig
	sshKeyByName *servers.SSHKey
}

func resourceServersServerV1UpdateLoadData(
	ctx context.Context, dsClient *servers.ServiceClient, d *schema.ResourceData,
	locationID, osID, configurationID, sshKeyName string,
) (*serversServerV1UpdateData, diag.Diagnostics) {
	operatingSystems, _, err := dsClient.OperatingSystems(ctx, servers.OperatingSystemsQuery{
		LocationID: locationID,
		ServiceID:  configurationID,
	})
	if err != nil {
		return nil, diag.FromErr(errGettingObjects(objectOS, err))
	}

	os := operatingSystems.FindOneByID(osID)

	if os == nil {
		return nil, diag.FromErr(errGettingObject(objectOS, osID, ErrNotFound))
	}

	partitionsConfigFromSchema, err := resourceServersServerV1ReadPartitionsConfig(d)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf(
			"failed to read partitions config: %w", err,
		))
	}

	var partitionsConfig servers.PartitionsConfig
	if !partitionsConfigFromSchema.IsEmpty() || os.Partitioning {
		if !os.Partitioning { // in case of configured partitions
			return nil, diag.FromErr(fmt.Errorf(
				"%s %s does not support partitions config", objectOS, os.OSValue,
			))
		}

		localDrives, _, err := dsClient.LocalDrives(ctx, configurationID)
		if err != nil {
			return nil, diag.FromErr(fmt.Errorf(
				"failed to get local drives for configuration %s: %w", configurationID, err,
			))
		}

		partitionsConfig, err = partitionsConfigFromSchema.CastToAPIPartitionsConfig(localDrives, os.DefaultPartitions)
		if err != nil {
			return nil, diag.FromErr(fmt.Errorf(
				"failed to read partitions config input: %w", err,
			))
		}
	}

	var sshKey *servers.SSHKey
	if sshKeyName != "" {
		sshKeys, _, err := dsClient.SSHKeys(ctx)
		if err != nil {
			return nil, diag.FromErr(fmt.Errorf(
				"failed to get SSH keys: %w", err,
			))
		}

		sshKey = sshKeys.FindOneByName(sshKeyName)
		if sshKey == nil {
			return nil, diag.FromErr(fmt.Errorf(
				"SSH key %s not found", sshKeyName,
			))
		}
	}

	return &serversServerV1UpdateData{
		os:           os,
		partitions:   partitionsConfig,
		sshKeyByName: sshKey,
	}, nil
}

func resourceServersServerV1UpdateValidatePreconditions(
	ctx context.Context, d *schema.ResourceData, dsClient *servers.ServiceClient,
	os *servers.OperatingSystem, partitions servers.PartitionsConfig,
	needUserScript, needSSHKey bool,
) diag.Diagnostics {
	var (
		osConfigChanged = d.HasChanges(serversServerSchemaKeyOSID) ||
			d.HasChange(serversServerSchemaKeyOSHostName) ||
			d.HasChange(serversServerSchemaKeyOSSSHKey) ||
			d.HasChange(serversServerSchemaKeyOSSSHKeyName) ||
			d.HasChange(serversServerSchemaKeyOSPassword) ||
			d.HasChange(serversServerSchemaKeyOSPartitionsConfig) ||
			d.HasChange(serversServerSchemaKeyOSUserScript)

		projectIDChanged       = d.HasChanges(serversServerSchemaKeyProjectID)
		locationIDChanged      = d.HasChanges(serversServerSchemaKeyLocationID)
		configurationIDChanged = d.HasChanges(serversServerSchemaKeyConfigurationID)
		pricePlanNameChanged   = d.HasChanges(serversServerSchemaKeyPricePlanName)

		osID = d.Get(serversServerSchemaKeyOSID).(string)
	)

	switch {
	case !osConfigChanged:
		return diag.Errorf("can't update cause os configuration has not changed")

	case projectIDChanged:
		prevID, _ := d.GetChange(serversServerSchemaKeyProjectID)

		return diag.Errorf("can't update case project ID has changed, use previous id %s", prevID)

	case locationIDChanged:
		prevID, _ := d.GetChange(serversServerSchemaKeyLocationID)

		return diag.Errorf("can't update case location ID has changed, use previous id %s", prevID)

	case configurationIDChanged:
		prevID, _ := d.GetChange(serversServerSchemaKeyConfigurationID)

		return diag.Errorf("can't update case configuration ID has changed, use previous id %s", prevID)

	case pricePlanNameChanged:
		prevName, _ := d.GetChange(serversServerSchemaKeyPricePlanName)

		return diag.Errorf("can't update case price plan ID has changed, use previous name %s", prevName)

	case needUserScript && !os.ScriptAllowed:
		return diag.FromErr(fmt.Errorf(
			"%s %s does not allow scripts", objectOS, osID,
		))

	case needSSHKey && !os.IsSSHKeyAllowed:
		return diag.FromErr(fmt.Errorf(
			"%s %s does not allow SSH keys", objectOS, osID,
		))

	case partitions != nil && os.OSValue == "windows":
		return diag.FromErr(fmt.Errorf(
			"%s %s does not support partitions config", objectOS, os.OSValue,
		))
	}

	configurationID := d.Get(serversServerSchemaKeyConfigurationID).(string)

	_, _, err := dsClient.PartitionsValidate(ctx, partitions, configurationID)
	if err != nil {
		return diag.FromErr(fmt.Errorf(
			"failed to validate partitions config: %w", err,
		))
	}

	return nil
}

func resourceServersServerV1ImportState(_ context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	config := meta.(*Config)
	if config.ProjectID == "" {
		return nil, errors.New("project_id must be set for the resource import")
	}

	_ = d.Set("project_id", config.ProjectID)

	return []*schema.ResourceData{d}, nil
}
