BUCKET ?= moneyman-artifacts

bucket:
	aws s3api create-bucket --bucket $(BUCKET) --create-bucket-configuration LocationConstraint=ap-southeast-2

package:
	sam package \
		--template sam-template.yml \
		--s3-bucket $(BUCKET) \
		--output-template-file packaged-template.yml

deploy: package
	sam deploy \
		--template packaged-template.yml \
		--s3-bucket $(BUCKET)