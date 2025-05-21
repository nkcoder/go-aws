package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create a readonly IAM group
		readOnlyGroup, err := iam.NewGroup(ctx, "readonly-group", &iam.GroupArgs{
			Name: pulumi.String("readonly"),
			Path: pulumi.String("/"),
		})
		if err != nil {
			return err
		}

		// Attach the AWS managed ReadOnly policy to the group
		// This policy provides readonly access to all AWS services
		_, err = iam.NewGroupPolicyAttachment(ctx, "readonly-policy-attachment", &iam.GroupPolicyAttachmentArgs{
			Group:     readOnlyGroup.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/ReadOnlyAccess"),
		})
		if err != nil {
			return err
		}

		// Create a new IAM user
		readOnlyUser, err := iam.NewUser(ctx, "readonly-user", &iam.UserArgs{
			Name: pulumi.String("readonly-user"),
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
		// In production, you would use a more secure way to generate and manage passwords
		loginProfile, err := iam.NewUserLoginProfile(ctx, "readonly-user-login", &iam.UserLoginProfileArgs{
			User:                  readOnlyUser.Name,
			PgpKey:                pulumi.String(""), // Empty PGP key means plaintext password
			PasswordLength:        pulumi.Int(16),    // Optional: specify password length if not providing password
			PasswordResetRequired: pulumi.Bool(true), // Force password change on first login
		})
		if err != nil {
			return err
		}

		// Export the user and login information
		ctx.Export("readonlyUserName", readOnlyUser.Name)
		ctx.Export("initialPassword", loginProfile.Password) // This will be the encrypted password if PGP is used
		ctx.Export("passwordResetRequired", loginProfile.PasswordResetRequired)

		return nil
	})
}
