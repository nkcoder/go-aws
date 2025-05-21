package main

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		conf := config.New(ctx, "")

		// Get configuration values
		groupName := conf.Require("groupName")
		userName := conf.Require("userName")
		passwordLength := conf.RequireInt("passwordLength")
		if passwordLength < 16 {
			passwordLength = 16
		}
		passwordResetRequired := conf.RequireBool("passwordResetRequired")

		// Create the group if it doesn't exist
		readOnlyGroup, err := createGroupIfNotExists(ctx, groupName)
		if err != nil {
			return err
		}

		// Create a new IAM user
		readOnlyUser, err := iam.NewUser(ctx, "readonly-user", &iam.UserArgs{
			Name: pulumi.String(userName),
			Path: pulumi.String("/"),
		})
		if err != nil {
			return err
		}

		// Add the user to the readonly group
		_, err = iam.NewGroupMembership(ctx, "readonly-group-membership", &iam.GroupMembershipArgs{
			Group: readOnlyGroup.Name,
			Users: pulumi.StringArray{readOnlyUser.Name},
		})
		if err != nil {
			return err
		}

		// Enable console access by creating a login profile with a password
		loginProfile, err := iam.NewUserLoginProfile(ctx, "readonly-user-login", &iam.UserLoginProfileArgs{
			User:                  readOnlyUser.Name,
			PgpKey:                pulumi.String(""), // Empty PGP key means plaintext password
			PasswordLength:        pulumi.Int(passwordLength),
			PasswordResetRequired: pulumi.Bool(passwordResetRequired),
		})
		if err != nil {
			return err
		}

		loginUrl, err := getLoginUrl(ctx)
		if err != nil {
			return err
		}

		// Export the user and login information
		ctx.Export("readonlyGroupName", readOnlyGroup.Name)
		ctx.Export("readonlyUserName", readOnlyUser.Name)
		ctx.Export("initialPassword", loginProfile.Password)
		ctx.Export("passwordResetRequired", loginProfile.PasswordResetRequired)
		ctx.Export("loginUrl", pulumi.String(loginUrl))

		return nil
	})
}

func createGroupIfNotExists(ctx *pulumi.Context, groupName string) (*iam.Group, error) {
	// Check if the group already exists
	var readOnlyGroup *iam.Group
	var err error

	// Try to get the existing group by name
	existingGroup, err := iam.LookupGroup(ctx, &iam.LookupGroupArgs{
		GroupName: groupName,
	})

	if err != nil || existingGroup == nil {
		// Group doesn't exist, create it
		readOnlyGroup, err = iam.NewGroup(ctx, "readonly-group", &iam.GroupArgs{
			Name: pulumi.String(groupName),
			Path: pulumi.String("/"),
		})
		if err != nil {
			return nil, err
		}

		// Attach the AWS managed ReadOnly policy to the group
		// This policy provides readonly access to all AWS services
		_, err = iam.NewGroupPolicyAttachment(ctx, "readonly-policy-attachment", &iam.GroupPolicyAttachmentArgs{
			Group:     readOnlyGroup.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/ReadOnlyAccess"),
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Group exists, import it as a resource reference
		existingGroupID := existingGroup.Id
		ctx.Log.Info("Group already exists with ID: "+existingGroupID, nil)

		// Create a reference to the existing group
		var importErr error
		readOnlyGroup, importErr = iam.NewGroup(ctx, "readonly-group-ref", &iam.GroupArgs{
			Name: pulumi.String(groupName),
		}, pulumi.Import(pulumi.ID(groupName)))

		if importErr != nil {
			return nil, fmt.Errorf("failed to import group '%s': %w", groupName, importErr)
		}
	}

	return readOnlyGroup, nil
}

func getLoginUrl(ctx *pulumi.Context) (string, error) {
	callerIdentity, err := aws.GetCallerIdentity(ctx, nil)
	if err != nil {
		return "", err
	}

	loginUrl := fmt.Sprintf("https://%s.signin.aws.amazon.com/console", callerIdentity.AccountId)
	return loginUrl, nil
}
